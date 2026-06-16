package job

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/count/count"
	rediskey "ran-feed/app/rpc/count/internal/common/consts/redis"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	bigVRebuildInterval  = time.Hour
	bigVRebuildTimeout   = 30 * time.Second
	bigVRebuildBatchSize = 500
)

// renameScript 原子把临时集合换到正式 key 重建期间读方始终看到旧集合无空窗
const renameScript = `redis.call('RENAME', KEYS[1], KEYS[2]); return 1`

// BigVRebuildJob 周期全量重建全局大 V 集合 预热存量 兜底 CDC 漂移与 Redis 丢失
type BigVRebuildJob struct {
	svcCtx    *svc.ServiceContext
	countRepo repositories.CountValueRepository
	doneCh    chan struct{}
}

func NewBigVRebuildJob(svcCtx *svc.ServiceContext) *BigVRebuildJob {
	return &BigVRebuildJob{
		svcCtx:    svcCtx,
		countRepo: repositories.NewCountValueRepository(context.Background(), svcCtx.MysqlDb),
		doneCh:    make(chan struct{}),
	}
}

// Start 启动即重建一次预热存量 之后周期自愈 多实例并发重建数据幂等可接受
func (j *BigVRebuildJob) Start() {
	j.rebuild()
	ticker := time.NewTicker(bigVRebuildInterval)
	defer ticker.Stop()
	for {
		select {
		case <-j.doneCh:
			return
		case <-ticker.C:
			j.rebuild()
		}
	}
}

func (j *BigVRebuildJob) Stop() {
	close(j.doneCh)
}

func (j *BigVRebuildJob) rebuild() {
	ctx, cancel := context.WithTimeout(context.Background(), bigVRebuildTimeout)
	defer cancel()
	logger := logx.WithContext(ctx)

	ids, err := j.countRepo.ListTargetIDsByValueGte(int32(count.BizType_FOLLOWED), int32(count.TargetType_USER), rediskey.BigVFollowerThreshold)
	if err != nil {
		logger.Errorf("重建大 V 集合查询失败 err=%v", err)
		return
	}

	globalKey := rediskey.RedisFeedBigVGlobalKey
	tmpKey := rediskey.RedisFeedBigVGlobalRebuildKey

	// 无大 V 直接清空正式集合
	if len(ids) == 0 {
		if _, derr := j.svcCtx.Redis.DelCtx(ctx, globalKey); derr != nil {
			logger.Errorf("清空大 V 集合失败 err=%v", derr)
		}
		return
	}

	// 重建到临时 key 再原子 RENAME 切换
	if _, derr := j.svcCtx.Redis.DelCtx(ctx, tmpKey); derr != nil {
		logger.Errorf("清理大 V 重建临时集合失败 err=%v", derr)
		return
	}
	batch := make([]any, 0, bigVRebuildBatchSize)
	flush := func() bool {
		if len(batch) == 0 {
			return true
		}
		if _, derr := j.svcCtx.Redis.SaddCtx(ctx, tmpKey, batch...); derr != nil {
			logger.Errorf("重建大 V 集合 SADD 失败 err=%v", derr)
			return false
		}
		batch = batch[:0]
		return true
	}
	for _, id := range ids {
		batch = append(batch, strconv.FormatInt(id, 10))
		if len(batch) >= bigVRebuildBatchSize {
			if !flush() {
				return
			}
		}
	}
	if !flush() {
		return
	}

	if _, derr := j.svcCtx.Redis.EvalCtx(ctx, renameScript, []string{tmpKey, globalKey}); derr != nil {
		logger.Errorf("重建大 V 集合 RENAME 失败 err=%v", derr)
		return
	}
	logger.Infof("大 V 集合重建完成 count=%d", len(ids))
}
