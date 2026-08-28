package feedservicelogic

import (
	"context"
	"math"
	"ran-feed/app/rpc/content/internal/logic/publishbox"
	"strconv"
	"strings"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	followInboxKeepN = 5000

	// followInboxBuildFolloweesScanCap 同步构建 inbox 时扫描关注的上限
	followInboxBuildFolloweesScanCap = 5000

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
		publishBox:  publishbox.New(ctx, svcCtx),
	}
}

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

	// inbox 只装小用户 命中走 Redis 未命中同步构建 都只含小用户
	inboxKey := rediskey.BuildFollowInboxKey(userID)
	inboxItems, inboxHasMore, err := l.loadInboxSource(inboxKey, userID, cursorScore, cursorID, pageSize)
	if err != nil {
		return nil, err
	}

	// 大 V 永远读时 merge 推拉结合的拉 冷热路径统一
	var pool []scoredID
	var poolHasMore bool
	if bigVIDs := l.loadViewerBigVList(userID); len(bigVIDs) > 0 {
		pool, poolHasMore = l.fetchBigVContentIDs(bigVIDs, cursorScore, pageSize)
	}
	ids, hasMore, nextCursor := mergeScored(inboxItems, inboxHasMore, pool, poolHasMore, cursorScore, cursorID, pageSize)

	if len(ids) == 0 {
		return emptyFollowFeedRes(), nil
	}

	// 走统一二级缓存取详情 关注流只读 PUBLIC 再旁挂作者与点赞
	details, err := l.resolver.resolveDetails(ids, true)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return emptyFollowFeedRes(), nil
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

// loadInboxSource 取 inbox 小号来源 命中走 Redis 未命中同步构建并返回首屏
func (l *FollowFeedLogic) loadInboxSource(inboxKey string, userID, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool, error) {
	items, hasMore, cacheExists, err := l.queryInboxIDs(inboxKey, cursorScore, pageSize)
	if err != nil {
		return nil, false, err
	}
	if cacheExists {
		return items, hasMore, nil
	}
	return l.buildInboxSync(inboxKey, userID, cursorScore, cursorID, pageSize)
}

// queryInboxIDs 原生读 inbox 窗口内当前页 候选 复合游标精确过滤交给 mergeScored
// 返回 候选 是否还有更多 缓存是否存在 空结果再 EXISTS 区分 key 不存在与窗口内无内容
func (l *FollowFeedLogic) queryInboxIDs(inboxKey string, cursorScore int64, pageSize int) ([]scoredID, bool, bool, error) {
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	maxScore := math.MaxFloat64
	if cursorScore > 0 {
		// inclusive 取到游标分 边界同分由 mergeScored 过滤
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

// buildInboxSync 同步构建 inbox 只取小号窗口内容 写缓存或空哨兵 返回本次首屏来源
func (l *FollowFeedLogic) buildInboxSync(inboxKey string, userID, cursorScore, cursorID int64, pageSize int) ([]scoredID, bool, error) {
	followees, err := l.listFolloweesCapped(userID, followInboxBuildFolloweesScanCap)
	if err != nil {
		return nil, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
	}
	small := l.filterSmallFollowees(followees)
	if len(small) == 0 {
		l.writeEmptyInboxSentinel(inboxKey)
		return nil, false, nil
	}

	statusPublished := int32(content.ContentStatus_PUBLISHED)
	visibilityPublic := int32(content.Visibility_PUBLIC)
	rows, err := l.contentRepo.ListFollowByAuthorsCursor(statusPublished, visibilityPublic, small, 0, followInboxKeepN)
	if err != nil {
		return nil, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注内容失败"))
	}
	if len(rows) == 0 {
		l.writeEmptyInboxSentinel(inboxKey)
		return nil, false, nil
	}

	// 全窗口回填缓存 后续翻页与读直接命中
	if werr := l.updateInboxCache(inboxKey, rows); werr != nil {
		l.Errorf("回填 inbox 缓存失败: %v", werr)
	}

	items, hasMore := pageScoredFromRows(rows, cursorScore, cursorID, pageSize)
	return items, hasMore, nil
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
func (l *FollowFeedLogic) writeEmptyInboxSentinel(inboxKey string) {
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	args := followwindow.WriteArgs(
		int64(followInboxKeepN),
		followwindow.CutoffMillis(days),
		followwindow.TTLSeconds(days),
		followwindow.NowMillis(),
		followInboxEmptySentinelID,
	)
	if _, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.UpdateFollowInboxZSetScript, []string{inboxKey}, args...); err != nil {
		l.Errorf("写空 inbox 哨兵失败 inboxKey=%s: %v", inboxKey, err)
	}
}

func (l *FollowFeedLogic) updateInboxCache(inboxKey string, rows []*model.RanFeedContent) error {
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
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
	args := followwindow.WriteArgs(int64(followInboxKeepN), followwindow.CutoffMillis(days), followwindow.TTLSeconds(days), pairs...)
	_, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.UpdateFollowInboxZSetScript, []string{inboxKey}, args...)
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
			ContentType:  content.ContentType(d.ContentType),
			AuthorId:     d.AuthorID,
			AuthorName:   authorName,
			AuthorAvatar: authorAvatar,
			Title:        d.Title,
			CoverUrl:     d.CoverURL,
			PublishedAt:  d.PublishedAt,
			IsLiked:      likedMap[d.ContentID],
			LikeCount:    likeCountMap[d.ContentID],
		})
	}
	return items
}
