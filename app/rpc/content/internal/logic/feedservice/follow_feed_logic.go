package feedservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	followInboxRebuildLockTTLSeconds = 30
	followInboxRebuildTimeout        = 20 * time.Second

	followInboxKeepN = 5000

	// rebuildFolloweesScanCap 异步重建 inbox 时扫描关注的上限
	rebuildFolloweesScanCap = 5000
	// coldBackfillFolloweesScanCap 同步冷兜底首屏扫描关注的上限
	coldBackfillFolloweesScanCap = 2000
)

type FollowFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	resolver    *contentDetailResolver
}

func NewFollowFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowFeedLogic {
	return &FollowFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		resolver:    newContentDetailResolver(ctx, svcCtx),
	}
}

func (l *FollowFeedLogic) FollowFeed(in *content.FollowFeedReq) (*content.FollowFeedRes, error) {
	// todo 查询该用户是否存在
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

	// 1. 先查 inbox 缓存
	inboxKey := rediskey.BuildFollowInboxKey(userID)
	items, nextCursor, hasMore, cacheExists, err := l.queryInboxIDs(inboxKey, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}

	var ids []int64
	if cacheExists {
		// 2a. 缓存命中：先 merge 大 V publish zset（推拉结合读时拉）
		ids = scoredIDsToIDs(items)
		bigVIDs := l.loadViewerBigVList(userID)
		if len(bigVIDs) > 0 {
			pool, anyMore := l.fetchBigVContentIDs(bigVIDs, in.Cursor, pageSize)
			if len(pool) > 0 {
				ids, hasMore, nextCursor = mergeScored(items, hasMore, pool, anyMore, pageSize)
			}
		}
	} else {
		// 2b. 缓存未命中：异步重建，同步走 DB 兜底返回首屏
		threading.GoSafe(func() {
			ctx, cancel := context.WithTimeout(context.Background(), followInboxRebuildTimeout)
			defer cancel()
			NewFollowFeedLogic(ctx, l.svcCtx).rebuildInboxWithLock(userID, inboxKey)
		})

		rows, more, cur, cerr := l.coldBackfill(userID, parseCursorID(in.Cursor), pageSize)
		if cerr != nil {
			return nil, cerr
		}
		ids = make([]int64, 0, len(rows))
		for _, r := range rows {
			if r != nil {
				ids = append(ids, r.ID)
			}
		}
		nextCursor = cur
		hasMore = more
	}

	if len(ids) == 0 {
		return emptyFollowFeedRes(), nil
	}

	// 3. 走统一二级缓存取详情 关注流只读 PUBLIC 再旁挂作者与点赞
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

func parseCursorID(cursor string) int64 {
	if cursor == "" || cursor == "0" {
		return 0
	}
	id, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
}

// rebuildInboxWithLock 加锁后重建 inbox 缓存，best-effort，错误只记日志
func (l *FollowFeedLogic) rebuildInboxWithLock(userID int64, inboxKey string) {
	lockKey := rediskey.BuildFollowInboxRebuildLockKey(userID)
	redisLock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
	redisLock.SetExpire(followInboxRebuildLockTTLSeconds)

	locked, err := redisLock.AcquireCtx(l.ctx)
	if err != nil {
		l.Errorf("获取重建锁失败 userID=%d: %v", userID, err)
		return
	}
	if !locked {
		return
	}
	defer func() {
		if ok, rerr := redisLock.ReleaseCtx(l.ctx); !ok || rerr != nil {
			l.Errorf("释放重建锁失败 userID=%d: %v", userID, rerr)
		}
	}()
	l.rebuildInboxCacheBestEffort(userID, inboxKey)
}

func (l *FollowFeedLogic) rebuildInboxCacheBestEffort(userID int64, inboxKey string) {
	followees, err := l.listFolloweesCapped(userID, rebuildFolloweesScanCap)
	if err != nil {
		l.Errorf("查询关注列表失败: %v", err)
		return
	}
	if len(followees) == 0 {
		return
	}

	statusPublished := int32(content.ContentStatus_PUBLISHED)
	visibilityPublic := int32(content.Visibility_PUBLIC)
	rows, qerr := l.contentRepo.ListFollowByAuthorsCursor(statusPublished, visibilityPublic, followees, 0, followInboxKeepN)
	if qerr != nil {
		l.Errorf("查询关注内容失败: %v", qerr)
		return
	}
	if len(rows) == 0 {
		return
	}
	if err = l.updateInboxCache(inboxKey, rows); err != nil {
		l.Errorf("回填缓存失败:%v", err)
	}
}

func (l *FollowFeedLogic) queryInboxIDs(inboxKey, cursor string, pageSize int) ([]scoredID, string, bool, bool, error) {
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryFollowInboxZSetScript,
		[]string{inboxKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
		strconv.FormatInt(followwindow.CutoffMillis(days), 10),
		strconv.Itoa(followwindow.TTLSeconds(days)),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注收件箱失败"))
	}
	items, nextCursor, hasMore, exists, ok := parseZSetReply(res)
	if !ok {
		return nil, "", false, false, errorx.NewMsg("查询关注收件箱失败")
	}
	return items, nextCursor, hasMore, exists, nil
}

func (l *FollowFeedLogic) coldBackfill(userID int64, cursorMillis int64, limit int) ([]*model.RanFeedContent, bool, string, error) {
	followees, err := l.listFolloweesCapped(userID, coldBackfillFolloweesScanCap)
	if err != nil {
		return nil, false, "", errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
	}
	if len(followees) == 0 {
		return nil, false, "", nil
	}

	statusPublished := int32(content.ContentStatus_PUBLISHED)
	visibilityPublic := int32(content.Visibility_PUBLIC)
	rows, err := l.contentRepo.ListFollowByAuthorsCursor(statusPublished, visibilityPublic, followees, cursorMillis, limit+1)
	if err != nil {
		return nil, false, "", errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注内容失败"))
	}

	hasMore := false
	if len(rows) > limit {
		hasMore = true
		rows = rows[:limit]
	}

	nextCursor := ""
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		if last.PublishedAt != nil {
			nextCursor = strconv.FormatInt(last.PublishedAt.UnixMilli(), 10)
		}
	}
	return rows, hasMore, nextCursor, nil
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
func buildFollowItems(details []*do.ContentDetailDO, userMap map[int64]*userservice.UserInfo, likedMap map[int64]bool, likeCountMap map[int64]int64) []*content.FollowFeedItem {
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
