package feedservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/interaction/client/followservice"
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
	ids, nextCursor, hasMore, cacheExists, err := l.queryInboxIDs(inboxKey, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}

	if cacheExists {
		// 2a. 缓存命中：先 merge 大 V publish zset（推拉结合读时拉）
		bigVIDs := l.loadViewerBigVList(userID)
		if len(bigVIDs) > 0 {
			pool, anyMore := l.fetchBigVContentIDs(bigVIDs, in.Cursor, pageSize)
			ids, hasMore, nextCursor = mergeContentIDs(ids, hasMore, pool, anyMore, pageSize)
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
	l.rebuildInboxCacheBestEffort(l.ctx, userID, inboxKey)
}

func (l *FollowFeedLogic) rebuildInboxCacheBestEffort(ctx context.Context, userID int64, inboxKey string) {
	followees, err := l.listFollowees(ctx, userID)
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

func (l *FollowFeedLogic) listFollowees(ctx context.Context, userID int64) ([]int64, error) {
	followees := make([]int64, 0)
	followCursor := int64(0)
	for {
		resp, err := l.svcCtx.FollowRpc.ListFollowees(ctx, &followservice.ListFolloweesReq{
			UserId:   userID,
			Cursor:   followCursor,
			PageSize: 500,
		})
		if err != nil {
			return nil, err
		}
		if resp != nil && len(resp.FollowUserIds) > 0 {
			followees = append(followees, resp.FollowUserIds...)
		}
		if resp == nil || !resp.HasMore || resp.NextCursor <= 0 {
			break
		}
		followCursor = resp.NextCursor
		if len(followees) >= 5000 {
			break
		}
	}
	return followees, nil
}

func (l *FollowFeedLogic) queryInboxIDs(inboxKey, cursor string, pageSize int) ([]int64, string, bool, bool, error) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryFollowInboxZSetScript,
		[]string{inboxKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注收件箱失败"))
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, "", false, false, errorx.NewMsg("查询关注收件箱失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	exists := existsVal == 1
	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1

	nextCursor := ""
	if hasMore {
		if s, ok := luaReplyString(arr[2]); ok {
			nextCursor = s
		}
	}

	ids := make([]int64, 0, len(arr)-3)
	for i := 3; i < len(arr); i++ {
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
	return ids, nextCursor, hasMore, exists, nil
}

func (l *FollowFeedLogic) coldBackfill(userID int64, cursorID int64, limit int) ([]*model.RanFeedContent, bool, string, error) {

	// 获取关注列表（分页拉取，避免一次返回过多）
	followees := make([]int64, 0)
	followCursor := int64(0)
	for {
		resp, err := l.svcCtx.FollowRpc.ListFollowees(l.ctx, &followservice.ListFolloweesReq{
			UserId:   userID,
			Cursor:   followCursor,
			PageSize: 200,
		})
		if err != nil {
			return nil, false, "", errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
		}
		if resp != nil && len(resp.FollowUserIds) > 0 {
			followees = append(followees, resp.FollowUserIds...)
		}
		if resp == nil || !resp.HasMore || resp.NextCursor <= 0 {
			break
		}
		followCursor = resp.NextCursor
		if len(followees) >= 2000 {
			break
		}
	}

	if len(followees) == 0 {
		return nil, false, "", nil
	}

	statusPublished := int32(content.ContentStatus_PUBLISHED)
	visibilityPublic := int32(content.Visibility_PUBLIC)
	rows, err := l.contentRepo.ListFollowByAuthorsCursor(statusPublished, visibilityPublic, followees, cursorID, limit+1)
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
		nextCursor = strconv.FormatInt(rows[len(rows)-1].ID, 10)
	}
	return rows, hasMore, nextCursor, nil
}

func (l *FollowFeedLogic) updateInboxCache(inboxKey string, rows []*model.RanFeedContent) error {
	keepN := int64(followInboxKeepN)
	args := make([]string, 0, 1+len(rows)*2)
	args = append(args, strconv.FormatInt(keepN, 10))
	for _, r := range rows {
		if r == nil {
			continue
		}
		idStr := strconv.FormatInt(r.ID, 10)
		args = append(args, idStr, idStr)
	}
	argsAny := make([]any, 0, len(args))
	for _, a := range args {
		argsAny = append(argsAny, a)
	}
	_, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.UpdateFollowInboxZSetScript, []string{inboxKey}, argsAny...)
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
