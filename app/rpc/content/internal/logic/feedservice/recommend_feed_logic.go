package feedservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type CacheResult int

const (
	CacheHit CacheResult = iota
	CacheMiss
	CacheError
)

type hotFeedResult struct {
	ids                []int64
	nextCursor         int64
	hasMore            bool
	resolvedSnapshotID string
}

type RecommendFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	resolver *contentDetailResolver
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		resolver: newContentDetailResolver(ctx, svcCtx),
	}
}

func (l *RecommendFeedLogic) RecommendFeed(in *content.RecommendFeedReq) (*content.RecommendFeedRes, error) {
	pageSize := int(in.PageSize)

	// 解析快照id和快照key
	preferredKey, preferredSnapshotID := l.resolveSnapshotKey(in.SnapshotId)

	// 从 Redis取id
	res, err := l.queryHotIDsByCursor(preferredKey, preferredSnapshotID, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if len(res.ids) == 0 {
		return &content.RecommendFeedRes{
			Items:      []*content.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: res.resolvedSnapshotID,
		}, nil
	}

	userID := int64(0)
	if in.UserId != nil {
		userID = *in.UserId
	}
	// 热榜只读 PUBLIC 走统一二级缓存
	items, err := l.resolver.assembleItems(res.ids, userID, true)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return &content.RecommendFeedRes{
			Items:      nil,
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: res.resolvedSnapshotID,
		}, nil
	}

	return &content.RecommendFeedRes{
		Items:      items,
		NextCursor: res.nextCursor,
		HasMore:    res.hasMore,
		SnapshotId: res.resolvedSnapshotID,
	}, nil
}

func (l *RecommendFeedLogic) resolveSnapshotKey(reqSnapshotID *string) (string, string) {
	if reqSnapshotID == nil || *reqSnapshotID == "" {
		return "", ""
	}
	return rediskey.BuildHotFeedSnapshotKey(*reqSnapshotID), *reqSnapshotID
}

func (l *RecommendFeedLogic) queryHotIDsByCursor(preferredKey, preferredSnapshotID string, cursorID string, pageSize int) (*hotFeedResult, error) {
	res, cacheResult := l.queryFromRedis(preferredKey, preferredSnapshotID, cursorID, pageSize)
	if cacheResult == CacheHit {
		return res, nil
	}
	return nil, errorx.NewMsg("热榜缓存不存在")
}

func (l *RecommendFeedLogic) queryFromRedis(preferredKey, preferredSnapshotID, cursor string, pageSize int) (*hotFeedResult, CacheResult) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryHotFeedZSetScript,
		[]string{
			preferredKey,
			rediskey.RedisFeedHotGlobalLatestKey,
			rediskey.RedisFeedHotGlobalSnapshotPrefix,
			rediskey.RedisFeedHotGlobalKey,
		},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
		preferredSnapshotID,
	)
	if err != nil {
		l.Errorf("Lua脚本执行失败: %v", err)
		return nil, CacheError
	}

	parsed, exists, parseErr := parseHotFeedLuaResult(res)
	if parseErr != nil {
		l.Errorf("解析Lua返回值失败: %v", parseErr)
		return nil, CacheError
	}
	if !exists {
		return nil, CacheMiss
	}
	return parsed, CacheHit
}

func parseHotFeedLuaResult(res any) (*hotFeedResult, bool, error) {
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 4 {
		return nil, false, errorx.NewMsg("查询热榜索引失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	exists := existsVal == 1

	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1

	nextCursor := int64(0)
	if hasMore {
		nextStr, _ := luaReplyString(arr[2])
		if nextStr != "" {
			v, parseErr := strconv.ParseInt(nextStr, 10, 64)
			if parseErr != nil {
				return nil, false, errorx.NewMsg("查询热榜索引失败")
			}
			nextCursor = v
		}
	}

	resolvedSnapshotID, _ := luaReplyString(arr[3])

	ids := make([]int64, 0, len(arr)-4)
	for i := 4; i < len(arr); i++ {
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

	return &hotFeedResult{
		ids:                ids,
		nextCursor:         nextCursor,
		hasMore:            hasMore,
		resolvedSnapshotID: resolvedSnapshotID,
	}, exists, nil
}
