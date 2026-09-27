package publishbox

import (
	"context"
	"errors"
	"math"
	"strconv"

	"ran-feed/app/rpc/content/internal/common/component/followwindow"
	"ran-feed/app/rpc/content/internal/common/component/redislock"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// emptySentinelID 空发件箱的负缓存哨兵 读时按 id<=0 过滤掉
	emptySentinelID int64 = 0
)

// ScoredID 发件箱 zset 的成员 ID 是 content_id Score 是发布时间毫秒
type ScoredID struct {
	ID    int64
	Score int64
}

// windowPage 一次窗口查询的结果
type windowPage struct {
	items   []ScoredID
	hasMore bool
}

// Box 作者发件箱 按发布时间倒序存作者已发布的内容 读命中直读 未命中抢锁回源重建
type Box struct {
	redis       *redis.Redis
	locker      *cache.DistLocker
	contentRepo repositories.ContentRepository
}

func New(
	redisClient *redis.Redis,
	locker *cache.DistLocker,
	contentRepo repositories.ContentRepository,
) *Box {
	return &Box{
		redis:       redisClient,
		locker:      locker,
		contentRepo: contentRepo,
	}
}

// QueryWindow 读作者发件箱一页 cutoffMillis 为 0 表示不限时间 作者主页要看全量历史 关注流只取窗口内的
func (b *Box) QueryWindow(ctx context.Context, authorID, cutoffMillis, cursorScore int64, pageSize int) ([]ScoredID, bool, error) {
	if authorID <= 0 || pageSize <= 0 {
		return nil, false, nil
	}
	feedKey := rediskey.BuildUserPublishFeedKey(authorID)

	page, err := cache.DoWithLock[windowPage](
		b.locker,
		ctx,
		cache.BuildLockKey(feedKey),
		func(ctx context.Context) (windowPage, bool, error) {
			items, hasMore, exists, qerr := b.queryOnce(ctx, feedKey, cutoffMillis, cursorScore, pageSize)
			if qerr != nil {
				return windowPage{}, false, qerr
			}
			return windowPage{
				items:   items,
				hasMore: hasMore,
			}, exists, nil
		},
		func(ctx context.Context) (windowPage, error) {
			return b.rebuild(ctx, feedKey, authorID, cutoffMillis, cursorScore, pageSize)
		},
	)
	if err != nil {
		// 等锁超时 本轮该作者缺席 不阻塞整条流
		if errors.Is(err, cache.ErrLockBusy) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return page.items, page.hasMore, nil
}

// queryOnce 只读缓存不重建 第三个返回值表示缓存 key 在不在 不在就让 DoWithLock 走重建
func (b *Box) queryOnce(ctx context.Context, feedKey string, cutoffMillis, cursorScore int64, pageSize int) ([]ScoredID, bool, bool, error) {
	maxScore := math.MaxFloat64
	if cursorScore > 0 {
		maxScore = float64(cursorScore)
	}
	pairs, err := b.redis.ZrevrangebyscoreWithScoresByFloatAndLimitCtx(
		ctx, feedKey, float64(cutoffMillis), maxScore, 0, pageSize+1)
	if err != nil {
		return nil, false, false, errorx.Wrap(ctx, err, errorx.NewMsg("查询发件箱失败"))
	}
	if len(pairs) == 0 {
		exists, eerr := b.redis.ExistsCtx(ctx, feedKey)
		if eerr != nil {
			return nil, false, false, errorx.Wrap(ctx, eerr, errorx.NewMsg("查询发件箱失败"))
		}
		return nil, false, exists, nil
	}
	if eerr := b.redis.ExpireCtx(ctx, feedKey, followwindow.TTLSeconds()); eerr != nil {
		logx.WithContext(ctx).Errorf("续期发件箱 TTL 失败 feedKey=%s: %v", feedKey, eerr)
	}
	items, hasMore := pairsToScored(pairs, pageSize)
	return items, hasMore, true, nil
}

// rebuild 回源重建发件箱 作者一条内容都没有就写空哨兵 否则全量回填后重读一次本页
func (b *Box) rebuild(ctx context.Context, feedKey string, authorID, cutoffMillis, cursorScore int64, pageSize int) (windowPage, error) {
	bgCtx := context.WithoutCancel(ctx)
	rows, err := b.contentRepo.ListPublishedByAuthorWithinWindow(ctx, authorID, 0, int(contentconsts.TimelineKeepN))
	if err != nil {
		return windowPage{}, errorx.Wrap(bgCtx, err, errorx.NewMsg("查询发件箱内容失败"))
	}
	if len(rows) == 0 {
		b.writeEmptySentinel(bgCtx, feedKey)
		return windowPage{}, nil
	}
	if werr := b.writeCache(bgCtx, feedKey, rows); werr != nil {
		logx.WithContext(ctx).Errorf("回填发件箱缓存失败 feedKey=%s: %v", feedKey, werr)
	}
	items, hasMore, _, qerr := b.queryOnce(bgCtx, feedKey, cutoffMillis, cursorScore, pageSize)
	if qerr != nil {
		return windowPage{}, qerr
	}
	return windowPage{items: items, hasMore: hasMore}, nil
}

// writeEmptySentinel 空作者写个哨兵占位做负缓存 免得每次读都回源查库
func (b *Box) writeEmptySentinel(ctx context.Context, feedKey string) {
	args := followwindow.WriteArgs(contentconsts.TimelineKeepN, 0, followwindow.TTLSeconds(), followwindow.NowMillis(), emptySentinelID)
	if _, err := b.redis.EvalCtx(ctx, redislock.UpdateUserPublishZSetScript, []string{feedKey}, args...); err != nil {
		logx.WithContext(ctx).Errorf("写空发件箱哨兵失败 feedKey=%s: %v", feedKey, err)
	}
}

// writeCache 全量回填发件箱 不按时间裁剪 只留最近 keepN 条 其余交给 TTL 过期
func (b *Box) writeCache(ctx context.Context, feedKey string, rows []*model.RanFeedContent) error {
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
	args := followwindow.WriteArgs(contentconsts.TimelineKeepN, 0, followwindow.TTLSeconds(), pairs...)
	_, err := b.redis.EvalCtx(ctx, redislock.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

// pairsToScored 解析 zset 成员并剔除哨兵与非法 id 多取的那一条只用来判断 hasMore
func pairsToScored(pairs []redis.FloatPair, pageSize int) ([]ScoredID, bool) {
	hasMore := len(pairs) > pageSize
	items := make([]ScoredID, 0, len(pairs))
	for _, p := range pairs {
		id, err := strconv.ParseInt(p.Key, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		items = append(items, ScoredID{ID: id, Score: int64(p.Score)})
	}
	return items, hasMore
}
