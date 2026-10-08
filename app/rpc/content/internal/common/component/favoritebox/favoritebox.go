// Package favoritebox 负责用户收藏流 feed:user:favorite 的全部读写
// 数据源是 interaction 域的收藏列表 本域只做缓存投影 命中直读 未命中抢锁回源重建
package favoritebox

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// emptySentinel 空收藏负缓存哨兵成员 读时按 id<=0 过滤不可见
const emptySentinel = "0"

// Page 一页收藏结果 游标是收藏记录 id 严格递减
type Page struct {
	ContentIDs []int64
	NextCursor string
	HasMore    bool
}

// Box 用户收藏流
type Box struct {
	redis       *redis.Redis
	locker      *cache.DistLocker
	favoriteRpc favoriteservice.FavoriteService
}

func New(
	redisClient *redis.Redis,
	locker *cache.DistLocker,
	favoriteRpc favoriteservice.FavoriteService,
) *Box {
	return &Box{
		redis:       redisClient,
		locker:      locker,
		favoriteRpc: favoriteRpc,
	}
}

// Query 读收藏流一页 命中读缓存 未命中抢锁回源重建 等锁超时降级单页回源
func (b *Box) Query(ctx context.Context, ownerID int64, cursor string, pageSize int) (Page, error) {
	if ownerID <= 0 || pageSize <= 0 {
		return Page{}, nil
	}
	feedKey := rediskey.BuildUserFavoriteFeedKey(ownerID)
	cursorScore := parseCursor(cursor)

	page, err := cache.DoWithLock(b.locker, ctx, cache.BuildLockKey(feedKey),
		func(ctx context.Context) (Page, bool, error) {
			p, exists, qerr := b.queryOnce(ctx, feedKey, cursorScore, pageSize)
			if qerr != nil {
				return Page{}, false, qerr
			}
			return p, exists, nil
		},
		func(ctx context.Context) (Page, error) {
			return b.rebuild(ctx, feedKey, ownerID, cursorScore, pageSize)
		},
	)
	if err != nil {
		// 等锁超时 本轮降级单页回源 不返回空
		if errors.Is(err, cache.ErrLockBusy) {
			return b.pageFromSource(ctx, ownerID, cursorScore, pageSize)
		}
		return Page{}, err
	}
	return page, nil
}

// Invalidate 失效某用户收藏流缓存 收藏变更后由 interaction 域跨域触发
func (b *Box) Invalidate(ctx context.Context, ownerID int64) error {
	if ownerID <= 0 {
		return nil
	}
	_, err := b.redis.DelCtx(ctx, rediskey.BuildUserFavoriteFeedKey(ownerID))
	return err
}

// queryOnce 只读缓存不重建 多取一条判 hasMore 空结果再 EXISTS 区分 key 不存在与窗口内真空
func (b *Box) queryOnce(ctx context.Context, feedKey string, cursorScore int64, pageSize int) (Page, bool, error) {
	// 雪花 id 唯一 start 减一即排除游标本身实现 score < cursor
	start := int64(math.MaxInt64)
	if cursorScore > 0 {
		start = cursorScore - 1
	}
	pairs, err := b.redis.ZrevrangebyscoreWithScoresAndLimitCtx(ctx, feedKey, start, 0, 0, pageSize+1)
	if err != nil {
		return Page{}, false, errorx.Wrap(ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	if len(pairs) == 0 {
		exists, eerr := b.redis.ExistsCtx(ctx, feedKey)
		if eerr != nil {
			return Page{}, false, errorx.Wrap(ctx, eerr, errorx.NewMsg("查询收藏列表失败"))
		}
		return Page{}, exists, nil
	}
	if eerr := b.redis.ExpireCtx(ctx, feedKey, contentconsts.TimelineTTLSeconds()); eerr != nil {
		logx.WithContext(ctx).Errorf("续期收藏缓存 TTL 失败 feedKey=%s err=%v", feedKey, eerr)
	}

	hasMore := len(pairs) > pageSize
	if hasMore {
		pairs = pairs[:pageSize]
	}
	nextCursor := ""
	if hasMore && len(pairs) > 0 {
		nextCursor = strconv.FormatInt(pairs[len(pairs)-1].Score, 10)
	}
	return Page{ContentIDs: pairsToIDs(pairs), NextCursor: nextCursor, HasMore: hasMore}, true, nil
}

// rebuild 持锁回源 interaction 域收藏列表 写满缓存或空哨兵 再回读本页
// 用脱离请求取消的 ctx 保证重建写入不被请求结束打断
func (b *Box) rebuild(ctx context.Context, feedKey string, ownerID, cursorScore int64, pageSize int) (Page, error) {
	bgCtx := context.WithoutCancel(ctx)
	resp, err := b.favoriteRpc.QueryFavoriteList(bgCtx, &favoriteservice.QueryFavoriteListReq{
		UserId:   ownerID,
		Cursor:   0,
		PageSize: uint32(contentconsts.TimelineKeepN),
	})
	if err != nil {
		return Page{}, errorx.Wrap(bgCtx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	if resp == nil || len(resp.Items) == 0 {
		b.writeEmptySentinel(bgCtx, feedKey)
		return Page{}, nil
	}
	if werr := b.writeCache(bgCtx, feedKey, resp.Items); werr != nil {
		logx.WithContext(ctx).Errorf("回填收藏缓存失败 feedKey=%s err=%v", feedKey, werr)
	}

	page, _, qerr := b.queryOnce(bgCtx, feedKey, cursorScore, pageSize)
	if qerr != nil {
		return Page{}, qerr
	}
	return page, nil
}

// pageFromSource 等锁超时的降级路径 直接回源单页 不写缓存
func (b *Box) pageFromSource(ctx context.Context, ownerID, cursorScore int64, pageSize int) (Page, error) {
	resp, err := b.favoriteRpc.QueryFavoriteList(ctx, &favoriteservice.QueryFavoriteListReq{
		UserId:   ownerID,
		Cursor:   cursorScore,
		PageSize: uint32(pageSize),
	})
	if err != nil {
		return Page{}, errorx.Wrap(ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	if resp == nil {
		return Page{}, nil
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
	return Page{ContentIDs: ids, NextCursor: nextCursor, HasMore: resp.HasMore}, nil
}

// writeCache 全量回填收藏 zset score 取收藏记录 id 容量与 TTL 与发布流对齐
func (b *Box) writeCache(ctx context.Context, feedKey string, items []*favoriteservice.FavoriteItem) error {
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
	return b.redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(ctx, feedKey, members...)
		pipe.ZRemRangeByRank(ctx, feedKey, 0, -contentconsts.TimelineKeepN-1)
		pipe.Expire(ctx, feedKey, cacheTTL())
		return nil
	})
}

// writeEmptySentinel 写不可见哨兵做空收藏负缓存 避免空用户每次读都回源
func (b *Box) writeEmptySentinel(ctx context.Context, feedKey string) {
	err := b.redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(ctx, feedKey, redis.Z{Score: 0, Member: emptySentinel})
		pipe.Expire(ctx, feedKey, cacheTTL())
		return nil
	})
	if err != nil {
		logx.WithContext(ctx).Errorf("写空收藏哨兵失败 feedKey=%s err=%v", feedKey, err)
	}
}

func cacheTTL() time.Duration {
	return time.Duration(contentconsts.TimelineTTLSeconds()) * time.Second
}

func parseCursor(cursor string) int64 {
	if cursor == "" {
		return 0
	}
	v, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func pairsToIDs(pairs []redis.Pair) []int64 {
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
