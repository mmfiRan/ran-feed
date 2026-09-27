// Package hotupdate 推荐榜算分任务 增量与全量两种模式共用 hotfeed 组件的算分与写榜逻辑
package hotupdate

import (
	"context"
	"fmt"
	"time"

	"ran-feed/app/rpc/content/internal/common/component/hotfeed"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// 增量与全量注册为两个 handler 共用同一实现 由 mode 区分批次来源
const (
	HandlerNameIncrement = "hot.fast.update"
	HandlerNameFull      = "hot.cold.update"
)

// 全量模式幂等锁 TTL 秒 仅防多实例同时全量重建 不承担与增量互斥
const defaultFullLockTTL = 1800

// Mode 任务模式
type Mode int

const (
	// ModeIncrement 增量 只算脏集合里的内容
	ModeIncrement Mode = iota
	// ModeFull 全量 按窗口扫库重算
	ModeFull
)

type Job struct {
	redis *redis.Redis
	feed  *hotfeed.Feed
	mode  Mode
}

// Register 注册增量与全量两个 handler
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	register(ctx, executor, svcCtx, HandlerNameIncrement, ModeIncrement)
	register(ctx, executor, svcCtx, HandlerNameFull, ModeFull)
}

func register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext, name string, mode Mode) {
	job := &Job{
		redis: svcCtx.Redis,
		feed:  svcCtx.HotFeed,
		mode:  mode,
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

	empty, err := j.feed.IsMainEmpty(ctx)
	if err != nil {
		return "", fmt.Errorf("读主榜数量失败 %w", err)
	}
	if empty {
		logger.Info("热榜主榜为空 增量模式自动转全量重建")
		return j.runFull(ctx, p, logger)
	}

	dirty, err := j.feed.UpdateIncremental(ctx, p.Options)
	if err != nil {
		return "", err
	}
	logger.Infof("热榜增量完成 dirty=%d", dirty)
	return "ok", nil
}

// runFull 按窗口扫库重算写主榜再刷新快照 只加全量幂等锁防多实例并发重建
// 算分幂等 与增量对同一内容写入的都是基于当时真实计数的合理分值 故两者无需互斥
func (j *Job) runFull(ctx context.Context, p Params, logger logx.Logger) (string, error) {
	lock := redis.NewRedisLock(j.redis, rediskey.BuildHotFeedFullLockKey())
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

	// 窗口起点在批次入口固定一次 整批复用
	startTime := time.Now().UTC().Add(-time.Duration(p.WindowDays) * 24 * time.Hour)
	logger.Infof("热榜全量开始 windowDays=%d mainN=%d topN=%d halfLife=%.1f shards=%d",
		p.WindowDays, p.MainN, p.TopN, p.HalfLifeHours, p.Shards)

	if err = j.feed.Rebuild(ctx, p.Options, startTime); err != nil {
		return "", err
	}
	logger.Info("热榜全量完成")
	return "ok", nil
}
