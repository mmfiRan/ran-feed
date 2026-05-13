package feedservicelogic

import (
	"sort"
	"strconv"
	"sync"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/client/followservice"

	"github.com/zeromicro/go-zero/core/mr"
)

const (
	defaultBigVMergeMaxQuery      = 100
	defaultBigVCacheTTLSeconds    = 300
	defaultBigVFolloweesScanLimit = 500
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

// computeViewerBigVList 列出关注（上限 ScanLimit），并行调 count-rpc 筛大 V
func (l *FollowFeedLogic) computeViewerBigVList(viewerID int64, cfg bigVMergeConfig) ([]int64, error) {
	followees, err := l.listFolloweesCapped(viewerID, cfg.ScanLimit)
	if err != nil {
		return nil, err
	}
	if len(followees) == 0 {
		return nil, nil
	}

	var (
		mu     sync.Mutex
		bigVs  = make([]int64, 0)
		logger = l.Logger
	)
	mr.ForEach(func(source chan<- int64) {
		for _, uid := range followees {
			source <- uid
		}
	}, func(uid int64) {
		resp, err := l.svcCtx.CountRpc.GetCount(l.ctx, &count.GetCountReq{
			BizType:    count.BizType_FOLLOWED,
			TargetType: count.TargetType_USER,
			TargetId:   uid,
		})
		if err != nil {
			logger.Errorf("查询粉丝数失败 uid=%d: %v", uid, err)
			return
		}
		if resp == nil || resp.Value < cfg.Threshold {
			return
		}
		mu.Lock()
		bigVs = append(bigVs, uid)
		mu.Unlock()
	}, mr.WithWorkers(cfg.CountConcurrency))

	return bigVs, nil
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

// fetchBigVContentIDs 并行查询每个大 V 的 publish zset 当前游标窗口
// 返回所有命中的 contentID 并集（含重复）与任意源是否还有更多
func (l *FollowFeedLogic) fetchBigVContentIDs(bigVIDs []int64, cursor string, pageSize int) ([]int64, bool) {
	if len(bigVIDs) == 0 {
		return nil, false
	}

	var (
		mu         sync.Mutex
		pool       = make([]int64, 0, len(bigVIDs)*pageSize)
		anyHasMore bool
		logger     = l.Logger
	)
	pageSizeStr := strconv.FormatInt(int64(pageSize), 10)

	mr.ForEach(func(source chan<- int64) {
		for _, uid := range bigVIDs {
			source <- uid
		}
	}, func(uid int64) {
		feedKey := rediskey.BuildUserPublishFeedKey(uid)
		ids, hasMore, err := l.queryBigVPublishIDs(feedKey, cursor, pageSizeStr)
		if err != nil {
			logger.Errorf("查询大 V publish zset 失败 uid=%d: %v", uid, err)
			return
		}
		if len(ids) == 0 && !hasMore {
			return
		}
		mu.Lock()
		pool = append(pool, ids...)
		if hasMore {
			anyHasMore = true
		}
		mu.Unlock()
	}, mr.WithWorkers(defaultBigVCountConcurrency))

	return pool, anyHasMore
}

// queryBigVPublishIDs 复用 QueryUserPublishZSetScript，返回当前窗口 contentID 列表 + hasMore
func (l *FollowFeedLogic) queryBigVPublishIDs(feedKey, cursor, pageSizeStr string) ([]int64, bool, error) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserPublishZSetScript,
		[]string{feedKey},
		cursor,
		pageSizeStr,
	)
	if err != nil {
		return nil, false, err
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, false, nil
	}
	existsVal, _ := luaReplyInt64(arr[0])
	if existsVal != 1 {
		return nil, false, nil
	}
	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1
	ids := make([]int64, 0, len(arr)-3)
	for i := 3; i < len(arr); i++ {
		s, _ := luaReplyString(arr[i])
		if s == "" {
			continue
		}
		id, parseErr := strconv.ParseInt(s, 10, 64)
		if parseErr != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids, hasMore, nil
}

// mergeContentIDs 合并 inbox 与大 V 池，按 contentID desc 去重排序，截取 pageSize
// 返回：合并后 ids、整体 hasMore、下一页 cursor
func mergeContentIDs(inboxIDs []int64, inboxHasMore bool, bigVPool []int64, bigVHasMore bool, pageSize int) ([]int64, bool, string) {
	if len(bigVPool) == 0 {
		nextCursor := ""
		if inboxHasMore && len(inboxIDs) > 0 {
			nextCursor = strconv.FormatInt(inboxIDs[len(inboxIDs)-1], 10)
		}
		return inboxIDs, inboxHasMore, nextCursor
	}

	seen := make(map[int64]struct{}, len(inboxIDs)+len(bigVPool))
	merged := make([]int64, 0, len(inboxIDs)+len(bigVPool))
	for _, id := range inboxIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}
	for _, id := range bigVPool {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}

	sort.Slice(merged, func(i, j int) bool { return merged[i] > merged[j] })

	overflow := len(merged) > pageSize
	if overflow {
		merged = merged[:pageSize]
	}
	hasMore := inboxHasMore || bigVHasMore || overflow

	nextCursor := ""
	if hasMore && len(merged) > 0 {
		nextCursor = strconv.FormatInt(merged[len(merged)-1], 10)
	}
	return merged, hasMore, nextCursor
}