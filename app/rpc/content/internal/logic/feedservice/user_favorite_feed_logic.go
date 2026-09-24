package feedservicelogic

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// favoriteEmptySentinel 空收藏负缓存哨兵成员 读时按 id<=0 过滤不可见
const favoriteEmptySentinel = "0"

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

// favoritePage 一页收藏结果 与发布流共用的游标语义
type favoritePage struct {
	ids        []int64
	nextCursor string
	hasMore    bool
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
	page, err := l.queryPage(feedKey, in.UserId, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if len(page.ids) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	// 列表按被访问者(owner)取 点赞态按实际访问者(viewer)算 二者不能混用
	items, err := l.resolver.assembleItems(page.ids, favoriteViewerID(in), true)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		// P5 过滤后为空也返回原始游标 避免整页死内容导致翻页中断
		return &content.UserFavoriteFeedRes{Items: []*content.ContentItem{}, NextCursor: page.nextCursor, HasMore: page.hasMore}, nil
	}

	return &content.UserFavoriteFeedRes{
		Items:      items,
		NextCursor: page.nextCursor,
		HasMore:    page.hasMore,
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

// favoriteViewerID 取实际访问者 未登录或未传为 0 匿名
// 收藏列表按 owner(user_id) 取 点赞态必须按 viewer(viewer_id) 算
func favoriteViewerID(in *content.UserFavoriteFeedReq) int64 {
	if in == nil || in.ViewerId == nil {
		return 0
	}
	return *in.ViewerId
}

// queryPage 与发布流同一套机制 命中读缓存 未命中抢锁回源重建 等锁超时降级单页回源
func (l *UserFavoriteFeedLogic) queryPage(feedKey string, ownerID int64, cursor string, pageSize int) (favoritePage, error) {
	cursorScore := parseFavoriteCursor(cursor)
	page, err := cache.DoWithLock(l.svcCtx.FavoriteFeedRebuildLocker, l.ctx, cache.BuildLockKey(feedKey),
		func(ctx context.Context) (favoritePage, bool, error) {
			ids, next, hasMore, exists, qerr := l.queryOnce(ctx, feedKey, cursorScore, pageSize)
			if qerr != nil {
				return favoritePage{}, false, qerr
			}
			return favoritePage{ids: ids, nextCursor: next, hasMore: hasMore}, exists, nil
		},
		func(ctx context.Context) (favoritePage, error) {
			return l.rebuild(ctx, feedKey, ownerID, cursorScore, pageSize)
		},
	)
	if err != nil {
		// 等锁超时 本轮降级单页回源 不返回空
		if errors.Is(err, cache.ErrLockBusy) {
			return l.pageFromSource(ownerID, cursorScore, pageSize)
		}
		return favoritePage{}, err
	}
	return page, nil
}

// queryOnce 原生读缓存窗口内当前页 游标按收藏记录 id 严格递减 多取 1 条判 hasMore
// 空结果再 EXISTS 区分 key 不存在与窗口内真空
func (l *UserFavoriteFeedLogic) queryOnce(ctx context.Context, feedKey string, cursorScore int64, pageSize int) ([]int64, string, bool, bool, error) {
	// 雪花 id 唯一 start 减一即排除游标本身实现 score < cursor
	start := int64(math.MaxInt64)
	if cursorScore > 0 {
		start = cursorScore - 1
	}
	pairs, err := l.svcCtx.Redis.ZrevrangebyscoreWithScoresAndLimitCtx(ctx, feedKey, start, 0, 0, pageSize+1)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	if len(pairs) == 0 {
		exists, eerr := l.svcCtx.Redis.ExistsCtx(ctx, feedKey)
		if eerr != nil {
			return nil, "", false, false, errorx.Wrap(ctx, eerr, errorx.NewMsg("查询收藏列表失败"))
		}
		return nil, "", false, exists, nil
	}
	if eerr := l.svcCtx.Redis.ExpireCtx(ctx, feedKey, l.cacheTTLSeconds()); eerr != nil {
		l.Errorf("续期收藏缓存 TTL 失败 feedKey=%s: %v", feedKey, eerr)
	}

	hasMore := len(pairs) > pageSize
	if hasMore {
		pairs = pairs[:pageSize]
	}
	nextCursor := ""
	if hasMore && len(pairs) > 0 {
		nextCursor = strconv.FormatInt(pairs[len(pairs)-1].Score, 10)
	}
	return favoritePairsToIDs(pairs), nextCursor, hasMore, true, nil
}

// rebuild 持锁回源 interaction 域收藏列表 写满缓存或空哨兵 再回读本页
// 用脱离请求取消的 ctx 保证重建写入不被请求结束打断
func (l *UserFavoriteFeedLogic) rebuild(ctx context.Context, feedKey string, ownerID, cursorScore int64, pageSize int) (favoritePage, error) {
	bgCtx := context.WithoutCancel(ctx)
	resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteList(bgCtx, &favoriteservice.QueryFavoriteListReq{
		UserId:   ownerID,
		Cursor:   0,
		PageSize: uint32(contentconsts.TimelineKeepN),
	})
	if err != nil {
		return favoritePage{}, errorx.Wrap(bgCtx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	if resp == nil || len(resp.Items) == 0 {
		l.writeEmptySentinel(bgCtx, feedKey)
		return favoritePage{}, nil
	}
	if werr := l.writeCache(bgCtx, feedKey, resp.Items); werr != nil {
		l.Errorf("回填收藏缓存失败 feedKey=%s: %v", feedKey, werr)
	}

	ids, next, hasMore, _, qerr := l.queryOnce(bgCtx, feedKey, cursorScore, pageSize)
	if qerr != nil {
		return favoritePage{}, qerr
	}
	return favoritePage{ids: ids, nextCursor: next, hasMore: hasMore}, nil
}

// pageFromSource 等锁超时的降级路径 直接回源单页 不写缓存
func (l *UserFavoriteFeedLogic) pageFromSource(ownerID, cursorScore int64, pageSize int) (favoritePage, error) {
	resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteList(l.ctx, &favoriteservice.QueryFavoriteListReq{
		UserId:   ownerID,
		Cursor:   cursorScore,
		PageSize: uint32(pageSize),
	})
	if err != nil {
		return favoritePage{}, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	if resp == nil {
		return favoritePage{}, nil
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
	return favoritePage{ids: ids, nextCursor: nextCursor, hasMore: resp.HasMore}, nil
}

// writeCache 全量回填收藏 zset 容量与 TTL 与发布流对齐
func (l *UserFavoriteFeedLogic) writeCache(ctx context.Context, feedKey string, items []*favoriteservice.FavoriteItem) error {
	members := make([]redis.Z, 0, len(items))
	for _, it := range items {
		if it == nil || it.ContentId <= 0 {
			continue
		}
		members = append(members, redis.Z{Score: float64(it.FavoriteId), Member: strconv.FormatInt(it.ContentId, 10)})
	}
	if len(members) == 0 {
		return nil
	}
	return l.svcCtx.Redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(ctx, feedKey, members...)
		pipe.ZRemRangeByRank(ctx, feedKey, 0, -contentconsts.TimelineKeepN-1)
		pipe.Expire(ctx, feedKey, l.cacheTTL())
		return nil
	})
}

// writeEmptySentinel 写不可见哨兵做空收藏负缓存 避免空用户每次读都回源
func (l *UserFavoriteFeedLogic) writeEmptySentinel(ctx context.Context, feedKey string) {
	err := l.svcCtx.Redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(ctx, feedKey, redis.Z{Score: 0, Member: favoriteEmptySentinel})
		pipe.Expire(ctx, feedKey, l.cacheTTL())
		return nil
	})
	if err != nil {
		l.Errorf("写空收藏哨兵失败 feedKey=%s: %v", feedKey, err)
	}
}

// cacheTTLSeconds 收藏缓存 TTL 秒 与发布流共用窗口推导 TTL
func (l *UserFavoriteFeedLogic) cacheTTLSeconds() int {
	days := contentconsts.WindowDays
	return followwindow.TTLSeconds(days)
}

func (l *UserFavoriteFeedLogic) cacheTTL() time.Duration {
	return time.Duration(l.cacheTTLSeconds()) * time.Second
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
