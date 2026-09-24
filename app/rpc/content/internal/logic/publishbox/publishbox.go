package publishbox

import (
	"context"
	"errors"
	"math"
	"strconv"

	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// emptySentinelID 空发件箱负缓存哨兵成员
	emptySentinelID int64 = 0
)

// ScoredID 发件箱结构体
type ScoredID struct {
	ID    int64
	Score int64
}

// windowPage 一次窗口查询的结果
type windowPage struct {
	items   []ScoredID
	hasMore bool
}

// PublishBox 作者发件箱
type PublishBox struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewPublishBox(ctx context.Context, svcCtx *svc.ServiceContext) *PublishBox {
	return &PublishBox{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

// QueryWindow 读作者发件箱
func (b *PublishBox) QueryWindow(authorID, cutoffMillis, cursorScore int64, pageSize int) ([]ScoredID, bool, error) {
	if authorID <= 0 || pageSize <= 0 {
		return nil, false, nil
	}
	feedKey := rediskey.BuildUserPublishFeedKey(authorID)

	page, err := cache.DoWithLock[windowPage](
		b.svcCtx.PublishBoxRebuildLocker,
		b.ctx,
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
		// 等待重建超时 本轮该作者缺席 不阻塞整流
		if errors.Is(err, cache.ErrLockBusy) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return page.items, page.hasMore, nil
}

// queryOnce 读发件箱窗口
func (b *PublishBox) queryOnce(ctx context.Context, feedKey string, cutoffMillis, cursorScore int64, pageSize int) ([]ScoredID, bool, bool, error) {
	maxScore := math.MaxFloat64
	if cursorScore > 0 {
		maxScore = float64(cursorScore)
	}
	pairs, err := b.svcCtx.Redis.ZrevrangebyscoreWithScoresByFloatAndLimitCtx(
		ctx, feedKey, float64(cutoffMillis), maxScore, 0, pageSize+1)
	if err != nil {
		return nil, false, false, errorx.Wrap(ctx, err, errorx.NewMsg("查询发件箱失败"))
	}
	if len(pairs) == 0 {
		exists, eerr := b.svcCtx.Redis.ExistsCtx(ctx, feedKey)
		if eerr != nil {
			return nil, false, false, errorx.Wrap(ctx, eerr, errorx.NewMsg("查询发件箱失败"))
		}
		return nil, false, exists, nil
	}
	days := contentconsts.WindowDays
	if eerr := b.svcCtx.Redis.ExpireCtx(ctx, feedKey, followwindow.TTLSeconds(days)); eerr != nil {
		b.Errorf("续期发件箱 TTL 失败 feedKey=%s: %v", feedKey, eerr)
	}
	items, hasMore := pairsToScored(pairs, pageSize)
	return items, hasMore, true, nil
}

// rebuild 重建发信箱
func (b *PublishBox) rebuild(ctx context.Context, feedKey string, authorID, cutoffMillis, cursorScore int64, pageSize int) (windowPage, error) {
	bgCtx := context.WithoutCancel(ctx)
	rows, err := b.contentRepo.ListPublishedByAuthorWithinWindow(authorID, 0, int(contentconsts.TimelineKeepN))
	if err != nil {
		return windowPage{}, errorx.Wrap(bgCtx, err, errorx.NewMsg("查询发件箱内容失败"))
	}
	if len(rows) == 0 {
		b.writeEmptySentinel(bgCtx, feedKey)
		return windowPage{}, nil
	}
	if werr := b.writeCache(bgCtx, feedKey, rows); werr != nil {
		b.Errorf("回填发件箱缓存失败 feedKey=%s: %v", feedKey, werr)
	}
	items, hasMore, _, qerr := b.queryOnce(bgCtx, feedKey, cutoffMillis, cursorScore, pageSize)
	if qerr != nil {
		return windowPage{}, qerr
	}
	return windowPage{items: items, hasMore: hasMore}, nil
}

// writeEmptySentinel 写不可见哨兵做空发件箱负缓存 避免空作者每次读都回源
func (b *PublishBox) writeEmptySentinel(ctx context.Context, feedKey string) {
	days := contentconsts.WindowDays
	args := followwindow.WriteArgs(contentconsts.TimelineKeepN, 0, followwindow.TTLSeconds(days), followwindow.NowMillis(), emptySentinelID)
	if _, err := b.svcCtx.Redis.EvalCtx(ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...); err != nil {
		b.Errorf("写空发件箱哨兵失败 feedKey=%s: %v", feedKey, err)
	}
}

// writeCache 全量回填发件箱 publish zset 不按时间裁剪 cutoff=0 仅 keepN 与 TTL 控量
func (b *PublishBox) writeCache(ctx context.Context, feedKey string, rows []*model.RanFeedContent) error {
	days := contentconsts.WindowDays
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
	args := followwindow.WriteArgs(contentconsts.TimelineKeepN, 0, followwindow.TTLSeconds(days), pairs...)
	_, err := b.svcCtx.Redis.EvalCtx(ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

// pairsToScored 过滤非法 id 与哨兵
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
	if len(items) > pageSize {
		items = items[:pageSize]
	}
	return items, hasMore
}
