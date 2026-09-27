// Package hotfeed 全站热榜的算分 写榜与快照
// 增量与全量共用同一套算分与写榜逻辑 算分是幂等的时点重算
// 两模式对同一内容写入的都是基于当时真实计数的合理分值 谁后写谁生效 故无需互斥
package hotfeed

import (
	"context"
	"fmt"
	"time"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/pkg/hotrank"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Feed 全站热榜的主榜 快照与脏集合读写
type Feed struct {
	redis       *redis.Redis
	countRpc    counterservice.CounterService
	contentRepo repositories.ContentRepository
}

func New(redisClient *redis.Redis, countRpc counterservice.CounterService, contentRepo repositories.ContentRepository) *Feed {
	return &Feed{
		redis:       redisClient,
		countRpc:    countRpc,
		contentRepo: contentRepo,
	}
}

// IsMainEmpty 主榜是否为空 供调用方决定增量是否退化成全量重建
func (f *Feed) IsMainEmpty(ctx context.Context) (bool, error) {
	card, err := f.redis.ZcardCtx(ctx, rediskey.RedisFeedHotGlobalKey)
	if err != nil {
		return false, err
	}
	return card == 0, nil
}

// Rebuild 按窗口扫库重算写主榜再刷新快照
// 窗口起点由调用方给定 保证同一批次内口径一致
func (f *Feed) Rebuild(ctx context.Context, opts Options, startTime time.Time) error {
	opts, err := opts.Normalize()
	if err != nil {
		return err
	}
	if err = f.rebuildFromDB(ctx, f.calculator(opts), startTime, opts); err != nil {
		return fmt.Errorf("从 DB 重建热榜失败 %w", err)
	}
	if err = f.refreshSnapshot(ctx, opts.MainN, opts.TopN); err != nil {
		return fmt.Errorf("刷新快照失败 %w", err)
	}
	return nil
}

// UpdateIncremental 冻结脏集合收集脏 ID 回查计数总量算全分覆盖主榜 再刷新快照 最后清冻结桶
// 返回本轮处理的脏内容数
func (f *Feed) UpdateIncremental(ctx context.Context, opts Options) (int, error) {
	opts, err := opts.Normalize()
	if err != nil {
		return 0, err
	}
	logger := logx.WithContext(ctx)

	dirtyIDs, err := f.collectDirtyIDs(ctx, opts.Shards)
	if err != nil {
		return 0, fmt.Errorf("收集脏 ID 失败 %w", err)
	}
	logger.Infof("热榜增量开始 shards=%d dirty=%d", opts.Shards, len(dirtyIDs))

	if len(dirtyIDs) > 0 {
		if err = f.recomputeAndOverwrite(ctx, f.calculator(opts), dirtyIDs, opts.MainN, opts.BatchSize); err != nil {
			return 0, fmt.Errorf("回查算分覆盖主榜失败 %w", err)
		}
	}
	if err = f.refreshSnapshot(ctx, opts.MainN, opts.TopN); err != nil {
		return 0, fmt.Errorf("刷新快照失败 %w", err)
	}
	if err = f.cleanupProcShards(ctx, opts.Shards); err != nil {
		return 0, fmt.Errorf("清理冻结桶失败 %w", err)
	}
	return len(dirtyIDs), nil
}

func (f *Feed) calculator(opts Options) hotrank.AdditiveTime {
	return hotrank.AdditiveTime{
		Weights:       mergeWeights(opts.Weights),
		HalfLifeHours: opts.HalfLifeHours,
	}
}
