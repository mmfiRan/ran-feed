package followfeed

import (
	"context"
	"errors"
	"math"
	"strconv"

	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// inboxEmptySentinelID 空收件箱的负缓存哨兵 读时按 id<=0 过滤不可见
const inboxEmptySentinelID int64 = 0

// AddFanOut 扇出写一批 follower 的收件箱 整批一次 pipeline
// 写收件箱幂等可重放 任一步失败整批返回错误交调用方重试
func (f *Feed) AddFanOut(ctx context.Context, followerIDs []int64, contentID, publishedAtMillis int64) error {
	if contentID <= 0 || len(followerIDs) == 0 {
		return nil
	}
	args := f.inboxArgs(publishedAtMillis, contentID)
	return f.redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		for _, followerID := range followerIDs {
			if followerID <= 0 {
				continue
			}
			pipe.Eval(ctx, updateInboxZSetScript, []string{rediskey.BuildFollowInboxKey(followerID)}, args...)
		}
		return nil
	})
}

// Backfill 把一批内容灌进单个 follower 的收件箱 返回实际新增数 新关注时调用
func (f *Feed) Backfill(ctx context.Context, followerID int64, items []ScoredID) (int, error) {
	if followerID <= 0 || len(items) == 0 {
		return 0, nil
	}
	pairs := make([]int64, 0, len(items)*2)
	for _, it := range items {
		if it.ID <= 0 {
			continue
		}
		pairs = append(pairs, it.Score, it.ID)
	}
	if len(pairs) == 0 {
		return 0, nil
	}
	added, err := f.writeInbox(ctx, rediskey.BuildFollowInboxKey(followerID), pairs...)
	if err != nil {
		return 0, errorx.Wrap(ctx, err, errorx.NewMsg("回填关注收件箱失败"))
	}
	return int(added), nil
}

// RemoveContents 从 follower 收件箱摘掉指定内容 取关时调用
func (f *Feed) RemoveContents(ctx context.Context, followerID int64, contentIDs []int64) (int, error) {
	if followerID <= 0 || len(contentIDs) == 0 {
		return 0, nil
	}
	members := make([]any, 0, len(contentIDs))
	for _, id := range contentIDs {
		if id > 0 {
			members = append(members, strconv.FormatInt(id, 10))
		}
	}
	if len(members) == 0 {
		return 0, nil
	}
	removed, err := f.redis.ZremCtx(ctx, rediskey.BuildFollowInboxKey(followerID), members...)
	if err != nil {
		return 0, errorx.Wrap(ctx, err, errorx.NewMsg("清理关注收件箱失败"))
	}
	return removed, nil
}

// ensureInbox 收件箱在即就绪 否则抢锁重建一次 抢不到返回不就绪由调用方降级
func (f *Feed) ensureInbox(ctx context.Context, viewerID int64, pullAuthors []int64) (bool, error) {
	inboxKey := rediskey.BuildFollowInboxKey(viewerID)
	if f.keyExists(ctx, inboxKey) {
		return true, nil
	}
	ok, err := cache.DoWithLock(f.locker, ctx, cache.BuildLockKey(inboxKey),
		func(ctx context.Context) (bool, bool, error) {
			return true, f.keyExists(ctx, inboxKey), nil
		},
		func(ctx context.Context) (bool, error) {
			return true, f.rebuildInbox(ctx, inboxKey, viewerID, pullAuthors)
		},
	)
	if err != nil {
		// 等锁超时 本轮降级回源 不阻塞整流
		if errors.Is(err, cache.ErrLockBusy) {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

// rebuildInbox 按推侧作者扫库回填收件箱 推侧为空写哨兵做负缓存
// 用脱离请求取消的 ctx 保证写入不被请求结束打断
func (f *Feed) rebuildInbox(ctx context.Context, inboxKey string, viewerID int64, pullAuthors []int64) error {
	bgCtx := context.WithoutCancel(ctx)

	followees, err := f.listFolloweesCapped(bgCtx, viewerID, followeesScanLimit)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("查询关注列表失败"))
	}
	small := excludeIDs(followees, pullAuthors)
	if len(small) == 0 {
		f.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}

	rows, err := f.contentRepo.ListFollowByAuthorsCursor(
		bgCtx,
		contentEnum.ContentStatusPublished,
		contentEnum.VisibilityPublic,
		small, 0, int(contentconsts.TimelineKeepN),
	)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("查询关注内容失败"))
	}
	if len(rows) == 0 {
		f.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}

	pairs := make([]int64, 0, len(rows)*2)
	for _, r := range rows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		pairs = append(pairs, r.PublishedAt.UnixMilli(), r.ID)
	}
	if len(pairs) == 0 {
		f.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}
	_, err = f.writeInbox(bgCtx, inboxKey, pairs...)
	return err
}

// queryInbox 读收件箱窗口内本页候选 多取一条判 hasMore 复合游标的精确过滤交调用方
func (f *Feed) queryInbox(ctx context.Context, viewerID, cursorScore int64, pageSize int) ([]ScoredID, bool, error) {
	inboxKey := rediskey.BuildFollowInboxKey(viewerID)
	maxScore := math.MaxFloat64
	if cursorScore > 0 {
		maxScore = float64(cursorScore)
	}
	pairs, err := f.redis.ZrevrangebyscoreWithScoresByFloatAndLimitCtx(
		ctx, inboxKey, float64(contentconsts.WindowCutoffMillis()), maxScore, 0, pageSize+1)
	if err != nil {
		return nil, false, errorx.Wrap(ctx, err, errorx.NewMsg("查询关注收件箱失败"))
	}
	if len(pairs) == 0 {
		return nil, false, nil
	}
	if eerr := f.redis.ExpireCtx(ctx, inboxKey, contentconsts.TimelineTTLSeconds()); eerr != nil {
		f.logger(ctx).Errorf("续期收件箱 TTL 失败 inboxKey=%s err=%v", inboxKey, eerr)
	}
	hasMore := len(pairs) > pageSize
	return pairsToScored(pairs), hasMore, nil
}

// writeEmptyInboxSentinel 写不可见哨兵做空收件箱负缓存 避免空用户每次读都重扫库
func (f *Feed) writeEmptyInboxSentinel(ctx context.Context, inboxKey string) {
	if _, err := f.writeInbox(ctx, inboxKey, contentconsts.NowMillis(), inboxEmptySentinelID); err != nil {
		f.logger(ctx).Errorf("写空收件箱哨兵失败 inboxKey=%s err=%v", inboxKey, err)
	}
}

// writeInbox 调收件箱写 lua 返回实际新增数
func (f *Feed) writeInbox(ctx context.Context, inboxKey string, pairs ...int64) (int64, error) {
	res, err := f.redis.EvalCtx(ctx, updateInboxZSetScript, []string{inboxKey}, f.inboxArgs(pairs...)...)
	if err != nil {
		return 0, err
	}
	added, _ := res.(int64)
	return added, nil
}

// inboxArgs 组装收件箱写 lua 的 ARGV keepN cutoff ttl 后跟 score member 对
func (f *Feed) inboxArgs(pairs ...int64) []any {
	args := make([]any, 0, 3+len(pairs))
	args = append(args,
		strconv.FormatInt(contentconsts.TimelineKeepN, 10),
		strconv.FormatInt(contentconsts.WindowCutoffMillis(), 10),
		strconv.Itoa(contentconsts.TimelineTTLSeconds()),
	)
	for _, v := range pairs {
		args = append(args, strconv.FormatInt(v, 10))
	}
	return args
}

// keyExists 判 key 在不在 查询失败按不在处理 宁可重建不可漏内容
func (f *Feed) keyExists(ctx context.Context, key string) bool {
	exists, err := f.redis.ExistsCtx(ctx, key)
	if err != nil {
		f.logger(ctx).Errorf("检查关注流缓存存在性失败 key=%s err=%v", key, err)
		return false
	}
	return exists
}

func (f *Feed) logger(ctx context.Context) logx.Logger {
	return logx.WithContext(ctx)
}

// pairsToScored 解析 zset 成员 过滤非法 id 与哨兵
func pairsToScored(pairs []redis.FloatPair) []ScoredID {
	items := make([]ScoredID, 0, len(pairs))
	for _, p := range pairs {
		id, err := strconv.ParseInt(p.Key, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		items = append(items, ScoredID{ID: id, Score: int64(p.Score)})
	}
	return items
}

// rowsToScored 回源行转收件箱成员
func rowsToScored(rows []*model.RanFeedContent) []ScoredID {
	items := make([]ScoredID, 0, len(rows))
	for _, r := range rows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		items = append(items, ScoredID{ID: r.ID, Score: r.PublishedAt.UnixMilli()})
	}
	return items
}
