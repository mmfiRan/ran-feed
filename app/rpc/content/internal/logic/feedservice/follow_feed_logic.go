package feedservicelogic

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"ran-feed/app/rpc/content/content"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/logic/publishbox"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// followInboxEmptySentinelID 空 inbox 负缓存哨兵成员 读时按 id<=0 过滤不可见
	followInboxEmptySentinelID int64 = 0
)

type FollowFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	resolver    *contentDetailResolver
	publishBox  *publishbox.PublishBox
}

func NewFollowFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowFeedLogic {
	return &FollowFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		resolver:    newContentDetailResolver(ctx, svcCtx),
		publishBox:  publishbox.NewPublishBox(ctx, svcCtx),
	}
}

// FollowFeed 关注流 推侧读收件箱本页 拉侧读拉模式集的作者发件箱 合并去重排序
func (l *FollowFeedLogic) FollowFeed(in *content.FollowFeedReq) (*content.FollowFeedRes, error) {
	if in == nil {
		return emptyFollowFeedRes(), nil
	}
	userID := in.UserId
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	cursorScore, cursorID := parseCursor(in.Cursor)

	pushItems, pushHasMore, pullAuthors := l.loadFollowSources(userID, cursorScore, cursorID, pageSize)

	var pool []scoredID
	var poolHasMore bool
	if len(pullAuthors) > 0 {
		pool, poolHasMore = l.fetchPullContentIDs(pullAuthors, cursorScore, pageSize)
	}
	ids, hasMore, nextCursor := mergeScored(pushItems, pushHasMore, pool, poolHasMore, cursorScore, cursorID, pageSize)

	if len(ids) == 0 {
		return emptyFollowFeedRes(), nil
	}

	// 走统一二级缓存取详情 关注流只读 PUBLIC 再旁挂作者与点赞
	details, err := l.resolver.resolveDetails(ids, true)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		// P5 过滤后为空也返回原始游标 避免整页死内容导致翻页中断
		return &content.FollowFeedRes{Items: []*content.FollowFeedItem{}, NextCursor: nextCursor, HasMore: hasMore}, nil
	}
	userMap, likedMap, likeCountMap, err := l.resolver.loadAuthorsAndLikes(details, userID)
	if err != nil {
		return nil, err
	}

	return &content.FollowFeedRes{
		Items:      buildFollowItems(details, userMap, likedMap, likeCountMap),
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyFollowFeedRes() *content.FollowFeedRes {
	return &content.FollowFeedRes{
		Items:      []*content.FollowFeedItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

// loadFollowSources 取推侧本页与拉侧作者集
// 两半缓存就绪走缓存 缺失抢锁同一次重建 等待超时或读缓存失败降级直接回源 不返回空
func (l *FollowFeedLogic) loadFollowSources(userID, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool, []int64) {
	inboxKey := rediskey.BuildFollowInboxKey(userID)
	pullKey := rediskey.BuildFollowPullKey(userID)

	ready, err := l.ensureFollowCaches(inboxKey, pullKey, userID)
	if err != nil {
		l.Errorf("准备关注流缓存失败 viewerID=%d: %v", userID, err)
	}
	if !ready {
		return l.degradeFollowSources(userID, cursorScore, cursorID, pageSize)
	}

	items, hasMore, _, qerr := l.queryInboxIDs(inboxKey, cursorScore, pageSize)
	if qerr != nil {
		l.Errorf("查询关注收件箱失败 viewerID=%d: %v", userID, qerr)
		return l.degradeFollowSources(userID, cursorScore, cursorID, pageSize)
	}
	return items, hasMore, l.readPullAuthors(l.ctx, pullKey)
}

// ensureFollowCaches 两半缓存都在即就绪 否则抢锁同一次重建 抢不到返回不就绪由调用方降级
func (l *FollowFeedLogic) ensureFollowCaches(inboxKey, pullKey string, userID int64) (bool, error) {
	if l.followCacheExistsCtx(l.ctx, inboxKey) && l.followCacheExistsCtx(l.ctx, pullKey) {
		return true, nil
	}
	ok, err := cache.DoWithLock(l.svcCtx.FollowRebuildLocker, l.ctx, cache.BuildLockKey(inboxKey),
		func(ctx context.Context) (bool, bool, error) {
			return true, l.followCacheExistsCtx(ctx, inboxKey) && l.followCacheExistsCtx(ctx, pullKey), nil
		},
		func(ctx context.Context) (bool, error) {
			return true, l.rebuildFollowCaches(ctx, inboxKey, pullKey, userID)
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

func (l *FollowFeedLogic) followCacheExistsCtx(ctx context.Context, key string) bool {
	exists, err := l.svcCtx.Redis.ExistsCtx(ctx, key)
	if err != nil {
		l.Errorf("检查关注流缓存存在性失败 key=%s: %v", key, err)
		return false
	}
	return exists
}

// rebuildFollowCaches 一次重建同时产出拉模式集与收件箱两半 同一 TTL 同生共死
// 由结构保证推拉并集恒等于关注列表 不再依赖多处判定恰好一致
// 用脱离请求取消的 ctx 保证写入不被请求结束打断
func (l *FollowFeedLogic) rebuildFollowCaches(ctx context.Context, inboxKey, pullKey string, userID int64) error {
	bgCtx := context.WithoutCancel(ctx)
	ttl := contentconsts.PullSetTTLSeconds

	followees, err := l.listFolloweesCapped(bgCtx, userID, defaultFolloweesScanLimit)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("查询关注列表失败"))
	}
	if len(followees) == 0 {
		if werr := l.writePullAuthors(bgCtx, pullKey, nil, ttl); werr != nil {
			return werr
		}
		l.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}

	bigVs, err := l.pickBigVFollowees(bgCtx, followees)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("判定大 V 关注失败"))
	}
	// 先写拉侧集再写收件箱 两半同一 TTL
	if werr := l.writePullAuthors(bgCtx, pullKey, bigVs, ttl); werr != nil {
		return werr
	}

	small := excludeFollowees(followees, bigVs)
	if len(small) == 0 {
		l.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}

	repo := repositories.NewContentRepository(bgCtx, l.svcCtx.MysqlDb)
	rows, err := repo.ListFollowByAuthorsCursor(
		int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
		int32(content.Visibility_VISIBILITY_PUBLIC),
		small, 0, int(contentconsts.TimelineKeepN),
	)
	if err != nil {
		return errorx.Wrap(bgCtx, err, errorx.NewMsg("查询关注内容失败"))
	}
	if len(rows) == 0 {
		l.writeEmptyInboxSentinel(bgCtx, inboxKey)
		return nil
	}
	return l.updateInboxCache(bgCtx, inboxKey, rows)
}

// degradeFollowSources 抢锁超时或读缓存失败时的安全降级 直接回源出本页与拉侧作者集 不返回空
func (l *FollowFeedLogic) degradeFollowSources(userID, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool, []int64) {
	followees, err := l.listFolloweesCapped(l.ctx, userID, defaultFolloweesScanLimit)
	if err != nil {
		l.Errorf("降级查询关注列表失败 viewerID=%d: %v", userID, err)
		return nil, false, nil
	}
	bigVs, err := l.pickBigVFollowees(l.ctx, followees)
	if err != nil {
		l.Errorf("降级判定大 V 关注失败 viewerID=%d: %v", userID, err)
		return nil, false, nil
	}
	small := excludeFollowees(followees, bigVs)
	if len(small) == 0 {
		return nil, false, bigVs
	}

	rows, err := l.contentRepo.ListFollowByAuthorsCursor(
		int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
		int32(content.Visibility_VISIBILITY_PUBLIC),
		small, 0, int(contentconsts.TimelineKeepN),
	)
	if err != nil {
		l.Errorf("降级查询关注内容失败 viewerID=%d: %v", userID, err)
		return nil, false, bigVs
	}
	items, hasMore := pageScoredFromRows(rows, cursorScore, cursorID, pageSize)
	return items, hasMore, bigVs
}

// parseCursor 解析复合游标 score:id
func parseCursor(cursor string) (int64, int64) {
	if cursor == "" || cursor == "0" {
		return 0, 0
	}
	if i := strings.IndexByte(cursor, ':'); i >= 0 {
		score, e1 := strconv.ParseInt(cursor[:i], 10, 64)
		id, e2 := strconv.ParseInt(cursor[i+1:], 10, 64)
		if e1 != nil || e2 != nil || score <= 0 {
			return 0, 0
		}
		if id < 0 {
			id = 0
		}
		return score, id
	}
	score, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || score <= 0 {
		return 0, 0
	}
	return score, 0
}

// formatCursor 组装复合游标 score:id
func formatCursor(score, id int64) string {
	return strconv.FormatInt(score, 10) + ":" + strconv.FormatInt(id, 10)
}

// afterCursor 判断 s 是否严格排在游标之后 按 published_at desc 加 content_id desc 总序
func afterCursor(s scoredID, cursorScore, cursorID int64) bool {
	if cursorScore <= 0 {
		return true
	}
	if s.score != cursorScore {
		return s.score < cursorScore
	}
	return s.id < cursorID
}

// queryInboxIDs 原生读 inbox 窗口内当前页候选 复合游标精确过滤交给 mergeScored
// 返回 候选 是否还有更多 缓存是否存在 空结果再 EXISTS 区分 key 不存在与窗口内无内容
func (l *FollowFeedLogic) queryInboxIDs(inboxKey string, cursorScore int64, pageSize int) ([]scoredID, bool, bool, error) {
	days := contentconsts.WindowDays
	maxScore := math.MaxFloat64
	if cursorScore > 0 {
		maxScore = float64(cursorScore)
	}
	pairs, err := l.svcCtx.Redis.ZrevrangebyscoreWithScoresByFloatAndLimitCtx(
		l.ctx, inboxKey, float64(followwindow.CutoffMillis(days)), maxScore, 0, pageSize+1)
	if err != nil {
		return nil, false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注收件箱失败"))
	}
	if len(pairs) == 0 {
		exists, eerr := l.svcCtx.Redis.ExistsCtx(l.ctx, inboxKey)
		if eerr != nil {
			return nil, false, false, errorx.Wrap(l.ctx, eerr, errorx.NewMsg("查询关注收件箱失败"))
		}
		return nil, false, exists, nil
	}
	hasMore := len(pairs) > pageSize
	if err := l.svcCtx.Redis.ExpireCtx(l.ctx, inboxKey, followwindow.TTLSeconds(days)); err != nil {
		l.Errorf("续期 inbox TTL 失败 inboxKey=%s: %v", inboxKey, err)
	}
	return scoredPairsToItems(pairs), hasMore, true, nil
}

// pageScoredFromRows 在已按 published_at desc 加 id desc 排好的行内 按复合游标取一页 多取 1 条判 hasMore
func pageScoredFromRows(rows []*model.RanFeedContent, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool) {
	items := make([]scoredID, 0, pageSize+1)
	for _, r := range rows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		s := scoredID{id: r.ID, score: r.PublishedAt.UnixMilli()}
		if !afterCursor(s, cursorScore, cursorID) {
			continue
		}
		items = append(items, s)
		if len(items) > pageSize {
			break
		}
	}
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}
	return items, hasMore
}

// writeEmptyInboxSentinel 写不可见哨兵成员做空 inbox 负缓存 避免空用户每次读都重扫库
func (l *FollowFeedLogic) writeEmptyInboxSentinel(ctx context.Context, inboxKey string) {
	days := contentconsts.WindowDays
	args := followwindow.WriteArgs(
		contentconsts.TimelineKeepN,
		followwindow.CutoffMillis(days),
		followwindow.TTLSeconds(days),
		followwindow.NowMillis(),
		followInboxEmptySentinelID,
	)
	if _, err := l.svcCtx.Redis.EvalCtx(ctx, luautils.UpdateFollowInboxZSetScript, []string{inboxKey}, args...); err != nil {
		l.Errorf("写空 inbox 哨兵失败 inboxKey=%s: %v", inboxKey, err)
	}
}

func (l *FollowFeedLogic) updateInboxCache(ctx context.Context, inboxKey string, rows []*model.RanFeedContent) error {
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
	args := followwindow.WriteArgs(contentconsts.TimelineKeepN, followwindow.CutoffMillis(days), followwindow.TTLSeconds(days), pairs...)
	_, err := l.svcCtx.Redis.EvalCtx(ctx, luautils.UpdateFollowInboxZSetScript, []string{inboxKey}, args...)
	return err
}

// buildFollowItems 按 details 顺序把 L2 详情加作者加点赞组装成 FollowFeedItem
func buildFollowItems(details []*do.ContentDetailDO, userMap map[int64]*user.UserInfo, likedMap map[int64]bool, likeCountMap map[int64]int64) []*content.FollowFeedItem {
	items := make([]*content.FollowFeedItem, 0, len(details))
	for _, d := range details {
		authorName := ""
		authorAvatar := ""
		if u, ok := userMap[d.AuthorID]; ok && u != nil {
			authorName = u.Nickname
			authorAvatar = u.Avatar
		}
		items = append(items, &content.FollowFeedItem{
			ContentId:    d.ContentID,
			ContentType:  utils.ContentTypeValue(d.ContentType),
			AuthorId:     d.AuthorID,
			AuthorName:   authorName,
			AuthorAvatar: authorAvatar,
			Title:        d.Title,
			CoverUrl:     d.CoverURL,
			PublishedAt:  timestamppb.New(time.Unix(d.PublishedAt, 0)),
			IsLiked:      likedMap[d.ContentID],
			LikeCount:    likeCountMap[d.ContentID],
		})
	}
	return items
}
