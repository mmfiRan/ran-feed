package feedservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/repositories"
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
	nextCursor         string
	hasMore            bool
	resolvedSnapshotID string
}

type RecommendFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	resolver    *contentDetailResolver
	contentRepo repositories.ContentRepository
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		resolver:    newContentDetailResolver(ctx, svcCtx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
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
			NextCursor: "",
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
		// P5 过滤后为空也返回原始游标 避免整页死内容导致翻页中断
		return &content.RecommendFeedRes{
			Items:      nil,
			NextCursor: res.nextCursor,
			HasMore:    res.hasMore,
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
	// Redis 丢数据或主榜快照均缺失时按 hot_score 游标查库兜底 保证推荐流不整接口失败
	return l.queryFromDB(cursorID, pageSize)
}

// queryFromDB 兜底查询 已发布加公开加未删 按 hot_score 与 id 倒序 keyset 翻页
func (l *RecommendFeedLogic) queryFromDB(cursor string, pageSize int) (*hotFeedResult, error) {
	cursorID := int64(0)
	if v, err := strconv.ParseInt(cursor, 10, 64); err == nil && v > 0 {
		cursorID = v
	}

	cursorScore := 0.0
	if cursorID > 0 {
		// 取游标内容的分值定位翻页位置 取不到说明该内容已删 退化为首页
		score, err := l.contentRepo.GetHotScoreByID(cursorID)
		if err != nil {
			l.Errorf("兜底查询解析游标分值失败 退化为首页 cursorID=%d err=%v", cursorID, err)
			cursorID, cursorScore = 0, 0
		} else {
			cursorScore = score
		}
	}

	rows, err := l.contentRepo.ListRecommendByHotScoreCursor(
		int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
		int32(content.Visibility_VISIBILITY_PUBLIC),
		cursorScore,
		cursorID,
		pageSize+1,
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询推荐流失败"))
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		ids = append(ids, row.ID)
	}
	hasMore := len(ids) > pageSize
	if hasMore {
		ids = ids[:pageSize]
	}
	nextCursor := ""
	if hasMore && len(ids) > 0 {
		nextCursor = strconv.FormatInt(ids[len(ids)-1], 10)
	}
	return &hotFeedResult{ids: ids, nextCursor: nextCursor, hasMore: hasMore}, nil
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

	nextCursor := ""
	if hasMore {
		nextCursor, _ = luaReplyString(arr[2])
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
