package bigv_reconcile

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/count/count"
	rediskey "ran-feed/app/rpc/count/internal/common/consts/redis"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/logx"
)

const HandlerName = "bigv.reconcile"

// reconcileBatchSize Redis 集合分批 SADD 大小
const reconcileBatchSize = 500

// BigVReconcileJob 大 V 定时修正 复查粉丝数补 CDC 漏网晋升 再把 Redis 集合与表对齐
type BigVReconcileJob struct {
	svc       *svc.ServiceContext
	countRepo repositories.CountValueRepository
	bigVRepo  repositories.BigVRepository
	logx.Logger
}

// Register 注册大 V 定时修正任务
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &BigVReconcileJob{
		svc:       svcCtx,
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
		bigVRepo:  repositories.NewBigVRepository(ctx, svcCtx.MysqlDb),
		Logger:    logx.WithContext(ctx),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

// Run 先复查计数补晋升 再 Redis 与表对齐 多实例并发幂等可接受
func (j *BigVReconcileJob) Run(ctx context.Context, _ xxljob.TriggerParam) (string, error) {
	if err := j.promoteMissed(ctx); err != nil {
		return "", err
	}
	if err := j.syncRedisFromTable(ctx); err != nil {
		return "", err
	}
	return "ok", nil
}

// promoteMissed 复查当前粉丝数 把 ≥阈值却漏晋升的补进大 V 表 单个失败只记日志不阻断
func (j *BigVReconcileJob) promoteMissed(ctx context.Context) error {
	candidates, err := j.countRepo.ListTargetValuesByValueGte(
		int32(count.BizType_FOLLOWED),
		int32(count.TargetType_USER),
		rediskey.BigVFollowerThreshold,
	)
	if err != nil {
		return err
	}
	for _, row := range candidates {
		if row == nil || row.TargetID <= 0 {
			continue
		}
		if err := j.bigVRepo.Promote(row.TargetID, row.Value); err != nil {
			j.Errorf("大 V 修正补晋升失败 userID=%d err=%v", row.TargetID, err)
		}
	}
	return nil
}

// syncRedisFromTable 以大 V 表为真相源全量灌临时 key 再 RENAME 原子替换 表空则清空正式集合
func (j *BigVReconcileJob) syncRedisFromTable(ctx context.Context) error {
	ids, err := j.bigVRepo.ListAllUserIDs()
	if err != nil {
		return err
	}

	globalKey := rediskey.RedisFeedBigVGlobalKey
	tmpKey := rediskey.RedisFeedBigVGlobalRebuildKey

	if len(ids) == 0 {
		_, derr := j.svc.Redis.DelCtx(ctx, globalKey)
		return derr
	}

	if _, err := j.svc.Redis.DelCtx(ctx, tmpKey); err != nil {
		return err
	}
	batch := make([]any, 0, reconcileBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if _, err := j.svc.Redis.SaddCtx(ctx, tmpKey, batch...); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	for _, id := range ids {
		batch = append(batch, strconv.FormatInt(id, 10))
		if len(batch) >= reconcileBatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}

	_, err = j.svc.Redis.DoCtx(ctx, "RENAME", tmpKey, globalKey)
	return err
}
