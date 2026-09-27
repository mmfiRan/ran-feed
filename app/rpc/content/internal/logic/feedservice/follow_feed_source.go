package feedservicelogic

import (
	"math"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/followwindow"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/pkg/errorx"
)

// loadFollowSources 取推侧本页与拉侧作者集
// 两半缓存就绪走缓存 缺失抢锁同一次重建 等待超时或读缓存失败降级直接回源 不返回空
func (l *FollowFeedLogic) loadFollowSources(userID, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool, []int64) {
	inboxKey := rediskey.BuildFollowInboxKey(userID)
	pullKey := rediskey.BuildFollowPullKey(userID)

	ready, err := l.ensureFollowCaches(inboxKey, pullKey, userID)
	if err != nil {
		l.Errorf("准备关注流缓存失败 viewerID=%d: %v", userID, err)
	}
	if !ready {
		return l.degradeFollowSources(userID, cursorScore, cursorID, pageSize)
	}

	items, hasMore, _, qerr := l.queryInboxIDs(inboxKey, cursorScore, pageSize)
	if qerr != nil {
		l.Errorf("查询关注收件箱失败 viewerID=%d: %v", userID, qerr)
		return l.degradeFollowSources(userID, cursorScore, cursorID, pageSize)
	}
	return items, hasMore, l.readPullAuthors(l.ctx, pullKey)
}

// degradeFollowSources 抢锁超时或读缓存失败时的安全降级 直接回源出本页与拉侧作者集 不返回空
func (l *FollowFeedLogic) degradeFollowSources(userID, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool, []int64) {
	followees, err := l.listFolloweesCapped(l.ctx, userID, defaultFolloweesScanLimit)
	if err != nil {
		l.Errorf("降级查询关注列表失败 viewerID=%d: %v", userID, err)
		return nil, false, nil
	}
	bigVs, err := l.pickBigVFollowees(l.ctx, followees)
	if err != nil {
		l.Errorf("降级判定大 V 关注失败 viewerID=%d: %v", userID, err)
		return nil, false, nil
	}
	small := excludeFollowees(followees, bigVs)
	if len(small) == 0 {
		return nil, false, bigVs
	}

	rows, err := l.contentRepo.ListFollowByAuthorsCursor(l.ctx,
		int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
		int32(content.Visibility_VISIBILITY_PUBLIC),
		small, 0, int(contentconsts.TimelineKeepN),
	)
	if err != nil {
		l.Errorf("降级查询关注内容失败 viewerID=%d: %v", userID, err)
		return nil, false, bigVs
	}
	items, hasMore := pageScoredFromRows(rows, cursorScore, cursorID, pageSize)
	return items, hasMore, bigVs
}

// queryInboxIDs 原生读 inbox 窗口内当前页候选 复合游标精确过滤交给 mergeScored
// 返回 候选 是否还有更多 缓存是否存在 空结果再 EXISTS 区分 key 不存在与窗口内无内容
func (l *FollowFeedLogic) queryInboxIDs(inboxKey string, cursorScore int64, pageSize int) ([]scoredID, bool, bool, error) {
	maxScore := math.MaxFloat64
	if cursorScore > 0 {
		maxScore = float64(cursorScore)
	}
	pairs, err := l.redis.ZrevrangebyscoreWithScoresByFloatAndLimitCtx(
		l.ctx, inboxKey, float64(followwindow.CutoffMillis()), maxScore, 0, pageSize+1)
	if err != nil {
		return nil, false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注收件箱失败"))
	}
	if len(pairs) == 0 {
		exists, eerr := l.redis.ExistsCtx(l.ctx, inboxKey)
		if eerr != nil {
			return nil, false, false, errorx.Wrap(l.ctx, eerr, errorx.NewMsg("查询关注收件箱失败"))
		}
		return nil, false, exists, nil
	}
	hasMore := len(pairs) > pageSize
	if err := l.redis.ExpireCtx(l.ctx, inboxKey, followwindow.TTLSeconds()); err != nil {
		l.Errorf("续期 inbox TTL 失败 inboxKey=%s: %v", inboxKey, err)
	}
	return scoredPairsToItems(pairs), hasMore, true, nil
}

// pageScoredFromRows 在已按 published_at desc 加 id desc 排好的行内 按复合游标取一页 多取 1 条判 hasMore
func pageScoredFromRows(rows []*model.RanFeedContent, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool) {
	items := make([]scoredID, 0, pageSize+1)
	for _, r := range rows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		s := scoredID{id: r.ID, score: r.PublishedAt.UnixMilli()}
		if !afterCursor(s, cursorScore, cursorID) {
			continue
		}
		items = append(items, s)
		if len(items) > pageSize {
			break
		}
	}
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}
	return items, hasMore
}
