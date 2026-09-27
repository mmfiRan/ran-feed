package feedservicelogic

import (
	"context"
	"errors"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/followwindow"
	"ran-feed/app/rpc/content/internal/common/component/redislock"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/errorx"
)

const (
	// followInboxEmptySentinelID 空 inbox 负缓存哨兵成员 读时按 id<=0 过滤不可见
	followInboxEmptySentinelID int64 = 0
)

// ensureFollowCaches 两半缓存都在即就绪 否则抢锁同一次重建 抢不到返回不就绪由调用方降级
func (l *FollowFeedLogic) ensureFollowCaches(inboxKey, pullKey string, userID int64) (bool, error) {
	if l.followCacheExistsCtx(l.ctx, inboxKey) && l.followCacheExistsCtx(l.ctx, pullKey) {
		return true, nil
	}
	ok, err := cache.DoWithLock(l.followRebuildLocker, l.ctx, cache.BuildLockKey(inboxKey),
		func(ctx context.Context) (bool, bool, error) {
			return true, l.followCacheExistsCtx(ctx, inboxKey) && l.followCacheExistsCtx(ctx, pullKey), nil
		},
		func(ctx context.Context) (bool, error) {
			return true, l.rebuildFollowCaches(ctx, inboxKey, pullKey, userID)
		},
	)
	if err != nil {
		// 等锁超时 本轮降级回源 不阻塞整流
		if errors.Is(err, cache.ErrLockBusy) {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (l *FollowFeedLogic) followCacheExistsCtx(ctx context.Context, key string) bool {
	exists, err := l.redis.ExistsCtx(ctx, key)
	if err != nil {
		l.Errorf("检查关注流缓存存在性失败 key=%s: %v", key, err)
		return false
	}
	return exists
}

// rebuildFollowCaches 一次重建同时产出拉模式集与收件箱两半 同一 TTL 同生共死
// 由结构保证推拉并集恒等于关注列表 不再依赖多处判定恰好一致
// 用脱离请求取消的 ctx 保证写入不被请求结束打断
func (l *FollowFeedLogic) rebuildFollowCaches(ctx context.Context, inboxKey, pullKey string, userID int64) error {
	bgCtx := context.WithoutCancel(ctx)
	ttl := contentconsts.PullSetTTLSeconds

	followees, err := l.listFolloweesCapped(bgCtx, userID, defaultFolloweesScanLimit)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("查询关注列表失败"))
	}
	if len(followees) == 0 {
		if werr := l.writePullAuthors(bgCtx, pullKey, nil, ttl); werr != nil {
			return werr
		}
		l.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}

	bigVs, err := l.pickBigVFollowees(bgCtx, followees)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("判定大 V 关注失败"))
	}
	// 先写拉侧集再写收件箱 两半同一 TTL
	if werr := l.writePullAuthors(bgCtx, pullKey, bigVs, ttl); werr != nil {
		return werr
	}

	small := excludeFollowees(followees, bigVs)
	if len(small) == 0 {
		l.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}

	rows, err := l.contentRepo.ListFollowByAuthorsCursor(
		bgCtx,
		int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
		int32(content.Visibility_VISIBILITY_PUBLIC),
		small, 0, int(contentconsts.TimelineKeepN),
	)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("查询关注内容失败"))
	}
	if len(rows) == 0 {
		l.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}
	return l.updateInboxCache(bgCtx, inboxKey, rows)
}

// writeEmptyInboxSentinel 写不可见哨兵成员做空 inbox 负缓存 避免空用户每次读都重扫库
func (l *FollowFeedLogic) writeEmptyInboxSentinel(ctx context.Context, inboxKey string) {
	args := followwindow.WriteArgs(
		contentconsts.TimelineKeepN,
		followwindow.CutoffMillis(),
		followwindow.TTLSeconds(),
		followwindow.NowMillis(),
		followInboxEmptySentinelID,
	)
	if _, err := l.redis.EvalCtx(ctx, redislock.UpdateFollowInboxZSetScript, []string{inboxKey}, args...); err != nil {
		l.Errorf("写空 inbox 哨兵失败 inboxKey=%s: %v", inboxKey, err)
	}
}

func (l *FollowFeedLogic) updateInboxCache(ctx context.Context, inboxKey string, rows []*model.RanFeedContent) error {
	pairs := make([]int64, 0, len(rows)*2)
	for _, r := range rows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		pairs = append(pairs, r.PublishedAt.UnixMilli(), r.ID)
	}
	if len(pairs) == 0 {
		return nil
	}
	args := followwindow.WriteArgs(contentconsts.TimelineKeepN, followwindow.CutoffMillis(), followwindow.TTLSeconds(), pairs...)
	_, err := l.redis.EvalCtx(ctx, redislock.UpdateFollowInboxZSetScript, []string{inboxKey}, args...)
	return err
}
