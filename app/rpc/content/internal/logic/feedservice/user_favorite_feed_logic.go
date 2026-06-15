package feedservicelogic

import (
	"context"
	"math"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type UserFavoriteFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	resolver *contentDetailResolver
}

func NewUserFavoriteFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFavoriteFeedLogic {
	return &UserFavoriteFeedLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		resolver: newContentDetailResolver(ctx, svcCtx),
	}
}

func (l *UserFavoriteFeedLogic) UserFavoriteFeed(in *content.UserFavoriteFeedReq) (*content.UserFavoriteFeedRes, error) {
	if in == nil {
		return emptyUserFavoriteFeedRes(), nil
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("用户id不能<=0")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	feedKey := buildUserFavoriteFeedKey(in.UserId)
	ids, nextCursor, hasMore, err := l.loadPageIDs(feedKey, in.UserId, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	items, err := l.resolver.assembleItems(ids, in.UserId, true)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	return &content.UserFavoriteFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyUserFavoriteFeedRes() *content.UserFavoriteFeedRes {
	return &content.UserFavoriteFeedRes{
		Items:      []*content.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

func buildUserFavoriteFeedKey(userID int64) string {
	return rediskey.BuildUserFavoriteFeedKey(userID)
}

// loadPageIDs 收藏流 L1 热头部旁路缓存 页落在最新 capacity 条内走缓存 翻过头部回 DB
func (l *UserFavoriteFeedLogic) loadPageIDs(feedKey string, userID int64, cursor string, pageSize int) ([]int64, string, bool, error) {
	cursorScore := parseFavoriteCursor(cursor)

	if err := l.ensureHotHead(feedKey, userID); err != nil {
		// 热头部重建失败 降级直接回源 DB 不阻断
		l.Errorf("重建收藏热头部失败 降级回源 userID=%d err=%v", userID, err)
		return l.pageFromDB(userID, cursorScore, pageSize)
	}

	// 雪花 id 唯一 start 减一即排除游标本身实现 score < cursor
	start := int64(math.MaxInt64)
	if cursorScore > 0 {
		start = cursorScore - 1
	}
	pairs, err := l.svcCtx.Redis.ZrevrangebyscoreWithScoresAndLimitCtx(l.ctx, feedKey, start, 0, 0, pageSize+1)
	if err != nil {
		l.Errorf("查询收藏热头部失败 降级回源 userID=%d err=%v", userID, err)
		return l.pageFromDB(userID, cursorScore, pageSize)
	}

	// 整页落在热头部内 直接走缓存
	if len(pairs) == pageSize+1 {
		ids := favoritePairsToIDs(pairs[:pageSize])
		return ids, strconv.FormatInt(pairs[pageSize-1].Score, 10), true, nil
	}

	// 取到热头部尾部 判断是真末页还是越界需回 DB
	card, err := l.svcCtx.Redis.ZcardCtx(l.ctx, feedKey)
	if err != nil {
		l.Errorf("查询收藏热头部容量失败 降级回源 userID=%d err=%v", userID, err)
		return l.pageFromDB(userID, cursorScore, pageSize)
	}
	if card < rediskey.RedisUserFavoriteFeedCapacity {
		// 热头部即全部收藏 这是真末页
		return favoritePairsToIDs(pairs), "", false, nil
	}
	// 热头部已满 更老的收藏在 DB 整页回源
	return l.pageFromDB(userID, cursorScore, pageSize)
}

// ensureHotHead 热头部缺失则重建最新 capacity
func (l *UserFavoriteFeedLogic) ensureHotHead(feedKey string, userID int64) error {
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, feedKey)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteList(l.ctx, &favoriteservice.QueryFavoriteListReq{
		UserId:   userID,
		Cursor:   0,
		PageSize: uint32(rediskey.RedisUserFavoriteFeedCapacity),
	})
	if err != nil {
		return err
	}
	if resp == nil || len(resp.Items) == 0 {
		return nil
	}

	members := make([]redis.Z, 0, len(resp.Items))
	for _, it := range resp.Items {
		if it == nil || it.ContentId <= 0 {
			continue
		}
		members = append(members, redis.Z{
			Score:  float64(it.FavoriteId),
			Member: strconv.FormatInt(it.ContentId, 10),
		})
	}
	if len(members) == 0 {
		return nil
	}

	return l.svcCtx.Redis.PipelinedCtx(l.ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(l.ctx, feedKey, members...)
		pipe.ZRemRangeByRank(l.ctx, feedKey, 0, -int64(rediskey.RedisUserFavoriteFeedCapacity)-1)
		pipe.Expire(l.ctx, feedKey, time.Duration(rediskey.RedisUserFavoriteFeedExpireSeconds)*time.Second)
		return nil
	})
}

// pageFromDB 越过热头部时按游标直接回源收藏 RPC 单页查询
func (l *UserFavoriteFeedLogic) pageFromDB(userID int64, cursorScore int64, pageSize int) ([]int64, string, bool, error) {
	resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteList(l.ctx, &favoriteservice.QueryFavoriteListReq{
		UserId:   userID,
		Cursor:   cursorScore,
		PageSize: uint32(pageSize),
	})
	if err != nil {
		return nil, "", false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	if resp == nil {
		return nil, "", false, nil
	}

	ids := make([]int64, 0, len(resp.Items))
	for _, it := range resp.Items {
		if it == nil || it.ContentId <= 0 {
			continue
		}
		ids = append(ids, it.ContentId)
	}
	nextCursor := ""
	if resp.HasMore && resp.NextCursor > 0 {
		nextCursor = strconv.FormatInt(resp.NextCursor, 10)
	}
	return ids, nextCursor, resp.HasMore, nil
}

func parseFavoriteCursor(cursor string) int64 {
	if cursor == "" {
		return 0
	}
	v, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func favoritePairsToIDs(pairs []redis.Pair) []int64 {
	ids := make([]int64, 0, len(pairs))
	for _, p := range pairs {
		id, err := strconv.ParseInt(p.Key, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}
