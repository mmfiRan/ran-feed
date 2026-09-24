package feedservicelogic

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/logic/publishbox"
	"ran-feed/app/rpc/interaction/client/followservice"

	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// defaultFolloweesScanLimit 重建时扫描关注上限 工程常量 推拉两半共用单一上限
	// 两条上限不一致会造出既没推也不拉的黑洞
	defaultFolloweesScanLimit = 5000
	// defaultPullConcurrency 拉侧并发查询作者发件箱度 工程常量不下发配置
	defaultPullConcurrency = 16
)

// pickBigVFollowees 批量判定候选中哪些是全局大 V 单次 Lua 一 RTT 不整体 SMEMBERS
// 成本随候选数增长 支撑棘轮语义下集合单调增长
func (l *FollowFeedLogic) pickBigVFollowees(ctx context.Context, candidates []int64) ([]int64, error) {
	args := make([]any, 0, len(candidates))
	for _, id := range candidates {
		if id > 0 {
			args = append(args, strconv.FormatInt(id, 10))
		}
	}
	if len(args) == 0 {
		return nil, nil
	}

	res, err := l.svcCtx.Redis.EvalCtx(ctx, luautils.FilterBigVMembersScript, []string{rediskey.RedisFeedBigVGlobalKey}, args...)
	if err != nil {
		return nil, err
	}
	arr, ok := res.([]interface{})
	if !ok {
		return nil, nil
	}

	bigVs := make([]int64, 0, len(arr))
	for _, v := range arr {
		s, ok := luaReplyString(v)
		if !ok {
			continue
		}
		id, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil || id <= 0 {
			continue
		}
		bigVs = append(bigVs, id)
	}
	return bigVs, nil
}

// excludeFollowees 返回 all 中不在 excluded 里的元素 保序
func excludeFollowees(all, excluded []int64) []int64 {
	if len(all) == 0 {
		return nil
	}
	if len(excluded) == 0 {
		return all
	}
	excludedSet := make(map[int64]struct{}, len(excluded))
	for _, id := range excluded {
		excludedSet[id] = struct{}{}
	}
	out := make([]int64, 0, len(all))
	for _, id := range all {
		if _, ok := excludedSet[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

// readPullAuthors 读拉模式关注集 已剔除哨兵
func (l *FollowFeedLogic) readPullAuthors(ctx context.Context, pullKey string) []int64 {
	members, err := l.svcCtx.Redis.SmembersCtx(ctx, pullKey)
	if err != nil {
		l.Errorf("读拉模式关注集失败 pullKey=%s: %v", pullKey, err)
		return nil
	}
	return parseIDMembers(members)
}

// writePullAuthors 写拉模式关注集 空集写哨兵防穿透 与收件箱同一 TTL 同生共死
func (l *FollowFeedLogic) writePullAuthors(ctx context.Context, pullKey string, authorIDs []int64, ttlSeconds int) error {
	members := make([]any, 0, len(authorIDs)+1)
	if len(authorIDs) == 0 {
		members = append(members, rediskey.FollowPullEmptySentinel)
	} else {
		for _, id := range authorIDs {
			members = append(members, strconv.FormatInt(id, 10))
		}
	}
	return l.svcCtx.Redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, pullKey)
		pipe.SAdd(ctx, pullKey, members...)
		pipe.Expire(ctx, pullKey, time.Duration(ttlSeconds)*time.Second)
		return nil
	})
}

// parseIDMembers 解析集合成员为 id 跳过空串与哨兵
func parseIDMembers(members []string) []int64 {
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if m == "" || m == rediskey.FollowPullEmptySentinel {
			continue
		}
		id, err := strconv.ParseInt(m, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// listFolloweesCapped 按 limit 截断的 ListFollowees 分页拉取
func (l *FollowFeedLogic) listFolloweesCapped(ctx context.Context, viewerID int64, limit int) ([]int64, error) {
	followees := make([]int64, 0)
	cursor := int64(0)
	for len(followees) < limit {
		pageSize := uint32(500)
		if remain := limit - len(followees); remain < int(pageSize) {
			pageSize = uint32(remain)
		}
		resp, err := l.svcCtx.FollowRpc.ListFollowees(ctx, &followservice.ListFolloweesReq{
			UserId:   viewerID,
			Cursor:   cursor,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.FollowUserIds) == 0 {
			break
		}
		followees = append(followees, resp.FollowUserIds...)
		if !resp.HasMore || resp.NextCursor <= 0 {
			break
		}
		cursor = resp.NextCursor
	}
	if len(followees) > limit {
		followees = followees[:limit]
	}
	return followees, nil
}

// fetchPullContentIDs 并发查拉侧作者发件箱当前窗口 走 publishbox 命中读未命中回源重建
// 返回所有命中的 (contentID published_at) 并集（含重复 由 mergeScored 去重）与任意源是否还有更多
func (l *FollowFeedLogic) fetchPullContentIDs(pullAuthors []int64, cursorScore int64, pageSize int) ([]scoredID, bool) {
	if len(pullAuthors) == 0 {
		return nil, false
	}

	var (
		mu         sync.Mutex
		pool       = make([]scoredID, 0, len(pullAuthors)*pageSize)
		anyHasMore bool
		logger     = l.Logger
	)

	cutoff := followwindow.CutoffMillis(contentconsts.WindowDays)

	mr.ForEach(func(source chan<- int64) {
		for _, uid := range pullAuthors {
			source <- uid
		}
	}, func(uid int64) {
		items, hasMore, err := l.publishBox.QueryWindow(uid, cutoff, cursorScore, pageSize)
		if err != nil {
			logger.Errorf("查询拉侧发件箱失败 uid=%d: %v", uid, err)
			return
		}
		if len(items) == 0 && !hasMore {
			return
		}
		mu.Lock()
		pool = append(pool, bigVScored(items)...)
		if hasMore {
			anyHasMore = true
		}
		mu.Unlock()
	}, mr.WithWorkers(defaultPullConcurrency))

	return pool, anyHasMore
}

// bigVScored 把 publishbox 候选转本包 scoredID
func bigVScored(items []publishbox.ScoredID) []scoredID {
	out := make([]scoredID, 0, len(items))
	for _, it := range items {
		out = append(out, scoredID{id: it.ID, score: it.Score})
	}
	return out
}

// scoredPairsToItems 把 zset 范围查询的 (member score) 对转 scoredID 过滤非法 id 与哨兵(id<=0)
func scoredPairsToItems(pairs []redis.FloatPair) []scoredID {
	items := make([]scoredID, 0, len(pairs))
	for _, p := range pairs {
		id, err := strconv.ParseInt(p.Key, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		items = append(items, scoredID{id: id, score: int64(p.Score)})
	}
	return items
}

// scoredIDsToIDs 按当前顺序抽出 content_id 丢弃 score
func scoredIDsToIDs(items []scoredID) []int64 {
	ids := make([]int64, 0, len(items))
	for _, s := range items {
		ids = append(ids, s.id)
	}
	return ids
}

// mergeScored 合并推侧收件箱与拉侧池 按复合游标(score id)过滤去重排序 截取 pageSize
// 各源 inclusive 取到游标分 边界同分在此按 (score id) 精确剔除 返回 content_id 列表 hasMore 复合游标
func mergeScored(push []scoredID, pushHasMore bool, pullPool []scoredID, pullHasMore bool, cursorScore, cursorID int64, pageSize int) ([]int64, bool, string) {
	seen := make(map[int64]struct{}, len(push)+len(pullPool))
	merged := make([]scoredID, 0, len(push)+len(pullPool))
	appendUniq := func(src []scoredID) {
		for _, s := range src {
			if s.id <= 0 || !afterCursor(s, cursorScore, cursorID) {
				continue
			}
			if _, ok := seen[s.id]; ok {
				continue
			}
			seen[s.id] = struct{}{}
			merged = append(merged, s)
		}
	}
	appendUniq(push)
	appendUniq(pullPool)

	// 按 published_at 倒序 同分以 content_id 倒序保证游标稳定
	sortScored(merged)

	overflow := len(merged) > pageSize
	if overflow {
		merged = merged[:pageSize]
	}
	hasMore := pushHasMore || pullHasMore || overflow

	nextCursor := ""
	if hasMore && len(merged) > 0 {
		last := merged[len(merged)-1]
		nextCursor = formatCursor(last.score, last.id)
	}
	return scoredIDsToIDs(merged), hasMore, nextCursor
}

// sortScored 按 published_at 倒序 同分以 content_id 倒序 保证游标稳定
func sortScored(items []scoredID) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].id > items[j].id
	})
}
