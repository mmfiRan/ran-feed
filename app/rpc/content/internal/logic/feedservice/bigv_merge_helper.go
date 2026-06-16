package feedservicelogic

import (
	"sort"
	"strconv"
	"sync"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/interaction/client/followservice"

	"github.com/zeromicro/go-zero/core/mr"
)

const (
	defaultBigVMergeMaxQuery      = 100
	defaultBigVCacheTTLSeconds    = 300
	defaultBigVFolloweesScanLimit = 5000
	defaultBigVCountConcurrency   = 16
	defaultBigVMergeThreshold     = 5000
)

type bigVMergeConfig struct {
	Threshold        int64
	MaxQuery         int
	CacheTTLSeconds  int
	ScanLimit        int
	CountConcurrency int
}

func resolveBigVMergeConfig(cfg config.FollowFanOutConfig) bigVMergeConfig {
	r := bigVMergeConfig{
		Threshold:        cfg.BigVFollowerThreshold,
		MaxQuery:         cfg.BigVMergeMaxQuery,
		CacheTTLSeconds:  cfg.BigVCacheTTLSeconds,
		ScanLimit:        cfg.BigVFolloweesScanLimit,
		CountConcurrency: cfg.BigVCountConcurrency,
	}
	if r.Threshold <= 0 {
		r.Threshold = defaultBigVMergeThreshold
	}
	if r.MaxQuery <= 0 {
		r.MaxQuery = defaultBigVMergeMaxQuery
	}
	if r.CacheTTLSeconds <= 0 {
		r.CacheTTLSeconds = defaultBigVCacheTTLSeconds
	}
	if r.ScanLimit <= 0 {
		r.ScanLimit = defaultBigVFolloweesScanLimit
	}
	if r.CountConcurrency <= 0 {
		r.CountConcurrency = defaultBigVCountConcurrency
	}
	return r
}

// loadViewerBigVList 读 viewer 大 V 缓存；miss 时同步计算并回写
// 返回的列表已剔除 sentinel；错误只记日志，调用方按空列表处理
func (l *FollowFeedLogic) loadViewerBigVList(viewerID int64) []int64 {
	cfg := resolveBigVMergeConfig(l.svcCtx.Config.FollowFanOut)
	cacheKey := rediskey.BuildFollowBigVKey(viewerID)

	members, err := l.svcCtx.Redis.SmembersCtx(l.ctx, cacheKey)
	if err != nil {
		l.Errorf("读取大 V 缓存失败 viewerID=%d: %v", viewerID, err)
	}
	if len(members) > 0 {
		return parseBigVMembers(members)
	}

	ids, computeErr := l.computeViewerBigVList(viewerID, cfg)
	if computeErr != nil {
		l.Errorf("计算大 V 列表失败 viewerID=%d: %v", viewerID, computeErr)
		return nil
	}
	l.writeBigVCache(cacheKey, ids, cfg.CacheTTLSeconds)
	if len(ids) > cfg.MaxQuery {
		ids = ids[:cfg.MaxQuery]
	}
	return ids
}

func parseBigVMembers(members []string) []int64 {
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if m == "" || m == rediskey.FollowBigVEmptySentinel {
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

// computeViewerBigVList 列出关注与全局大 V 集合内存求交得 viewer 大 V
// 取代逐个 count-rpc 扫描 消除 N 次 RPC 与截断漏读 是否大 V 由 count 侧维护的全局集合判定
func (l *FollowFeedLogic) computeViewerBigVList(viewerID int64, cfg bigVMergeConfig) ([]int64, error) {
	followees, err := l.listFolloweesCapped(viewerID, cfg.ScanLimit)
	if err != nil {
		return nil, err
	}
	if len(followees) == 0 {
		return nil, nil
	}

	globalSet, err := l.loadGlobalBigVSet()
	if err != nil {
		return nil, err
	}
	if len(globalSet) == 0 {
		return nil, nil
	}

	bigVs := make([]int64, 0)
	for _, uid := range followees {
		if _, ok := globalSet[uid]; ok {
			bigVs = append(bigVs, uid)
		}
	}
	return bigVs, nil
}

// loadGlobalBigVSet 读全局大 V 集合到内存 集合体量受控 SMEMBERS 一次取回
func (l *FollowFeedLogic) loadGlobalBigVSet() (map[int64]struct{}, error) {
	members, err := l.svcCtx.Redis.SmembersCtx(l.ctx, rediskey.RedisFeedBigVGlobalKey)
	if err != nil {
		return nil, err
	}
	set := make(map[int64]struct{}, len(members))
	for _, m := range members {
		id, perr := strconv.ParseInt(m, 10, 64)
		if perr != nil || id <= 0 {
			continue
		}
		set[id] = struct{}{}
	}
	return set, nil
}

// listFolloweesCapped 按 limit 截断的 ListFollowees 分页拉取
func (l *FollowFeedLogic) listFolloweesCapped(viewerID int64, limit int) ([]int64, error) {
	followees := make([]int64, 0)
	cursor := int64(0)
	for len(followees) < limit {
		pageSize := uint32(500)
		if remain := limit - len(followees); remain < int(pageSize) {
			pageSize = uint32(remain)
		}
		resp, err := l.svcCtx.FollowRpc.ListFollowees(l.ctx, &followservice.ListFolloweesReq{
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

// writeBigVCache DEL + SADD + EXPIRE，非原子但 5min TTL 内偶发 race 可接受
func (l *FollowFeedLogic) writeBigVCache(cacheKey string, ids []int64, ttlSeconds int) {
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, cacheKey); err != nil {
		l.Errorf("清空大 V 缓存失败 key=%s: %v", cacheKey, err)
	}
	members := make([]any, 0, len(ids)+1)
	if len(ids) == 0 {
		members = append(members, rediskey.FollowBigVEmptySentinel)
	} else {
		for _, id := range ids {
			members = append(members, strconv.FormatInt(id, 10))
		}
	}
	if _, err := l.svcCtx.Redis.SaddCtx(l.ctx, cacheKey, members...); err != nil {
		l.Errorf("写入大 V 缓存失败 key=%s: %v", cacheKey, err)
		return
	}
	if err := l.svcCtx.Redis.ExpireCtx(l.ctx, cacheKey, ttlSeconds); err != nil {
		l.Errorf("设置大 V 缓存 TTL 失败 key=%s: %v", cacheKey, err)
	}
}

// fetchBigVContentIDs 并行查询每个大 V 的 publish zset 当前窗口
// 返回所有命中的 (contentID published_at) 并集（含重复）与任意源是否还有更多
func (l *FollowFeedLogic) fetchBigVContentIDs(bigVIDs []int64, cursor string, pageSize int) ([]scoredID, bool) {
	if len(bigVIDs) == 0 {
		return nil, false
	}

	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	pageSizeStr := strconv.FormatInt(int64(pageSize), 10)
	cutoffStr := strconv.FormatInt(followwindow.CutoffMillis(days), 10)
	ttlStr := strconv.Itoa(followwindow.TTLSeconds(days))

	var (
		mu         sync.Mutex
		pool       = make([]scoredID, 0, len(bigVIDs)*pageSize)
		anyHasMore bool
		logger     = l.Logger
	)

	mr.ForEach(func(source chan<- int64) {
		for _, uid := range bigVIDs {
			source <- uid
		}
	}, func(uid int64) {
		feedKey := rediskey.BuildUserPublishFeedKey(uid)
		items, hasMore, err := l.queryBigVPublishIDs(feedKey, cursor, pageSizeStr, cutoffStr, ttlStr)
		if err != nil {
			logger.Errorf("查询大 V publish zset 失败 uid=%d: %v", uid, err)
			return
		}
		if len(items) == 0 && !hasMore {
			return
		}
		mu.Lock()
		pool = append(pool, items...)
		if hasMore {
			anyHasMore = true
		}
		mu.Unlock()
	}, mr.WithWorkers(defaultBigVCountConcurrency))

	return pool, anyHasMore
}

// queryBigVPublishIDs 复用 QueryUserPublishZSetScript，返回当前窗口 (contentID score) 列表 + hasMore
func (l *FollowFeedLogic) queryBigVPublishIDs(feedKey, cursor, pageSizeStr, cutoffStr, ttlStr string) ([]scoredID, bool, error) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserPublishZSetScript,
		[]string{feedKey},
		cursor,
		pageSizeStr,
		cutoffStr,
		ttlStr,
	)
	if err != nil {
		return nil, false, err
	}
	items, _, hasMore, exists, ok := parseZSetReply(res)
	if !ok || !exists {
		return nil, false, nil
	}
	return items, hasMore, nil
}

// scoredIDsToIDs 按当前顺序抽出 content_id 丢弃 score
func scoredIDsToIDs(items []scoredID) []int64 {
	ids := make([]int64, 0, len(items))
	for _, s := range items {
		ids = append(ids, s.id)
	}
	return ids
}

// mergeScored 合并 inbox 与大 V 池，按 published_at desc 去重排序，截取 pageSize
// 返回：合并后 content_id 列表、整体 hasMore、下一页 cursor(末位 score)
func mergeScored(inbox []scoredID, inboxHasMore bool, bigVPool []scoredID, bigVHasMore bool, pageSize int) ([]int64, bool, string) {
	seen := make(map[int64]struct{}, len(inbox)+len(bigVPool))
	merged := make([]scoredID, 0, len(inbox)+len(bigVPool))
	for _, s := range inbox {
		if _, ok := seen[s.id]; ok {
			continue
		}
		seen[s.id] = struct{}{}
		merged = append(merged, s)
	}
	for _, s := range bigVPool {
		if _, ok := seen[s.id]; ok {
			continue
		}
		seen[s.id] = struct{}{}
		merged = append(merged, s)
	}

	// 按 published_at 倒序 同分以 content_id 倒序保证游标稳定
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].score != merged[j].score {
			return merged[i].score > merged[j].score
		}
		return merged[i].id > merged[j].id
	})

	overflow := len(merged) > pageSize
	if overflow {
		merged = merged[:pageSize]
	}
	hasMore := inboxHasMore || bigVHasMore || overflow

	nextCursor := ""
	if hasMore && len(merged) > 0 {
		nextCursor = strconv.FormatInt(merged[len(merged)-1].score, 10)
	}
	return scoredIDsToIDs(merged), hasMore, nextCursor
}