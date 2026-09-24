// Package hot_update 推荐榜算分任务 增量与全量两种模式共用同一套算分与写榜逻辑
// 算分是幂等的时点重算 两模式对同一内容写入的都是基于当时真实计数的合理分值 谁后写谁生效 故无需互斥
package hot_update

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/consts"
	"ran-feed/pkg/hotrank"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// 增量与全量注册为两个 handler 共用同一实现 由 mode 区分批次来源
const (
	HandlerNameIncrement = "hot.fast.update"
	HandlerNameFull      = "hot.cold.update"
)

const (
	// 主榜候选池大小 裁剪线 远大于对外快照 留候补垫扛删除侵蚀
	defaultMainN = 5000
	// 重建后取前 TopN 作为对外可查询快照
	defaultTopN = 2000
	// 全量模式幂等锁 TTL 秒 仅防多实例同时全量重建 不承担与增量互斥
	defaultFullLockTTL = 1800
	// 与增量共用的默认半衰期小时
	defaultHalfLifeHour = 24
	// 落库与回查计数批大小
	defaultBatchSize = 500
	// 全量扫库分页大小
	defaultPageSize = 1000
	// 冻结桶 SSCAN 每批拉取数量 防单个大 key 阻塞
	defaultScanBatch = 1000
	// 快照默认 1 小时过期 避免历史快照无限累积
	defaultSnapshotTTL = 3600
)

// Mode 任务模式
type Mode int

const (
	// ModeIncrement 增量 只算脏集合里的内容
	ModeIncrement Mode = iota
	// ModeFull 全量 按窗口扫库重算
	ModeFull
)

type Params struct {
	Shards        int              `json:"shards"`
	MainN         int              `json:"mainN"`
	TopN          int              `json:"topN"`
	LockTTL       int              `json:"lockTtl"`
	HalfLifeHours float64          `json:"halfLifeHours"`
	Weights       *hotrank.Weights `json:"weights"`
	BatchSize     int              `json:"batchSize"`
	PageSize      int              `json:"pageSize"`
	WindowDays    int              `json:"windowDays"`
}

type Job struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
	mode        Mode
}

// Register 注册增量与全量两个 handler
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	register(ctx, executor, svcCtx, HandlerNameIncrement, ModeIncrement)
	register(ctx, executor, svcCtx, HandlerNameFull, ModeFull)
}

func register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext, name string, mode Mode) {
	job := &Job{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		mode:        mode,
	}
	executor.RegisterTask(name, job.Run)
}

// Run 增量模式在主榜为空时自动转全量 分钟级自愈 Redis 丢数据
func (j *Job) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	p, err := parseParams(param.ExecutorParams)
	if err != nil {
		return "", err
	}
	logger := logx.WithContext(ctx)

	if j.mode == ModeFull {
		return j.runFull(ctx, p, logger)
	}

	card, err := j.svc.Redis.ZcardCtx(ctx, rediskey.RedisFeedHotGlobalKey)
	if err != nil {
		return "", fmt.Errorf("读主榜数量失败 %w", err)
	}
	if card == 0 {
		logger.Info("热榜主榜为空 增量模式自动转全量重建")
		return j.runFull(ctx, p, logger)
	}
	return j.runIncrement(ctx, p, logger)
}

// runIncrement 冻结脏集合收集脏 ID 回查计数总量算全分覆盖主榜 再刷新快照 最后清冻结桶
func (j *Job) runIncrement(ctx context.Context, p Params, logger logx.Logger) (string, error) {
	dirtyIDs, err := j.collectDirtyIDs(ctx, p.Shards)
	if err != nil {
		return "", fmt.Errorf("收集脏 ID 失败 %w", err)
	}
	logger.Infof("热榜增量开始 shards=%d dirty=%d", p.Shards, len(dirtyIDs))

	if len(dirtyIDs) > 0 {
		if err := j.recomputeAndOverwrite(ctx, j.calculator(p), dirtyIDs, p.MainN, p.BatchSize); err != nil {
			return "", fmt.Errorf("回查算分覆盖主榜失败 %w", err)
		}
	}
	if err := j.refreshSnapshot(ctx, p.MainN, p.TopN); err != nil {
		return "", fmt.Errorf("刷新快照失败 %w", err)
	}
	if err := j.cleanupProcShards(ctx, p.Shards); err != nil {
		return "", fmt.Errorf("清理冻结桶失败 %w", err)
	}

	logger.Infof("热榜增量完成 dirty=%d", len(dirtyIDs))
	return "ok", nil
}

// runFull 按窗口扫库重算写主榜再刷新快照 只加全量幂等锁防多实例并发重建
func (j *Job) runFull(ctx context.Context, p Params, logger logx.Logger) (string, error) {
	lock := redis.NewRedisLock(j.svc.Redis, rediskey.BuildHotFeedFullLockKey())
	lock.SetExpire(p.LockTTL)
	locked, err := lock.AcquireCtx(ctx)
	if err != nil {
		return "", fmt.Errorf("抢全量重建锁失败 %w", err)
	}
	if !locked {
		logger.Info("热榜全量放弃 已有实例在跑")
		return "busy", nil
	}
	defer func() {
		if ok, releaseErr := lock.ReleaseCtx(context.Background()); !ok || releaseErr != nil {
			logger.Errorf("释放全量重建锁失败 held=%v err=%v", ok, releaseErr)
		}
	}()

	now := time.Now().UTC()
	startTime := now.Add(-time.Duration(p.WindowDays) * 24 * time.Hour)
	logger.Infof("热榜全量开始 windowDays=%d mainN=%d topN=%d halfLife=%.1f shards=%d",
		p.WindowDays, p.MainN, p.TopN, p.HalfLifeHours, p.Shards)

	if err := j.rebuildFromDB(ctx, j.calculator(p), startTime, now, p); err != nil {
		return "", fmt.Errorf("从 DB 重建热榜失败 %w", err)
	}
	if err := j.refreshSnapshot(ctx, p.MainN, p.TopN); err != nil {
		return "", fmt.Errorf("刷新快照失败 %w", err)
	}

	logger.Info("热榜全量完成")
	return "ok", nil
}

// parseParams 解析 XXL-JOB 参数 缺省填默认值 分片数与写入方不一致直接报错
func parseParams(raw string) (Params, error) {
	p := Params{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	if p.Shards <= 0 {
		p.Shards = consts.HotDirtyShards
	}
	// 分片数必须与 count 侧写入分片一致 否则部分分片永远不被处理 不静默跑一半
	if p.Shards != consts.HotDirtyShards {
		return Params{}, fmt.Errorf("热榜分片数 %d 与约定 %d 不一致", p.Shards, consts.HotDirtyShards)
	}
	if p.TopN <= 0 {
		p.TopN = defaultTopN
	}
	if p.MainN <= 0 {
		p.MainN = defaultMainN
	}
	// 主榜候选池必须 >= 对外快照 否则没有候补垫 退化成主榜=快照
	if p.MainN < p.TopN {
		p.MainN = p.TopN
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultFullLockTTL
	}
	if p.BatchSize <= 0 {
		p.BatchSize = defaultBatchSize
	}
	if p.PageSize <= 0 {
		p.PageSize = defaultPageSize
	}
	if p.HalfLifeHours <= 0 {
		p.HalfLifeHours = defaultHalfLifeHour
	}
	// 窗口缺省时从半衰期推导 与竞争视界对齐 显式传 windowDays 不覆盖
	if p.WindowDays <= 0 {
		p.WindowDays = deriveWindowDays(p.HalfLifeHours)
	}
	return p, nil
}

func (j *Job) calculator(p Params) hotrank.AdditiveTime {
	return hotrank.AdditiveTime{
		Weights:       mergeWeights(p.Weights),
		HalfLifeHours: p.HalfLifeHours,
	}
}
