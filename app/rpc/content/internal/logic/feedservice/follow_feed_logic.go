package feedservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/stores/redis"
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
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewFollowFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowFeedLogic {
	return &FollowFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *FollowFeedLogic) FollowFeed(in *content.FollowFeedReq) (*content.FollowFeedRes, error) {

	// todo 查询该用户是否存在
	if in == nil {
		return &content.FollowFeedRes{
			Items:      []*content.FollowFeedItem{},
			NextCursor: "",
			HasMore:    false,
		}, nil
	}
	userID := in.UserId
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	// 优先查询用户收信箱
	inboxKey := rediskey.BuildFollowInboxKey(userID)
	ids, nextCursor, hasMore, cacheExists, err := l.queryInboxIDs(inboxKey, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}

	// 仅当缓存不存在时，才会回填缓存
	// 缓存存在但本次返回数量不足，视为已到末尾，直接返回即可
	if !cacheExists {
		lockKey := rediskey.BuildFollowInboxRebuildLockKey(userID)
		redisLock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
		redisLock.SetExpire(followInboxRebuildLockTTLSeconds)
		locked, lockErr := redisLock.AcquireCtx(l.ctx)
		if lockErr != nil {
			return nil, errorx.Wrap(l.ctx, lockErr, errorx.NewMsg("查询失败请稍后重试"))
		}
		if locked {
			defer func() {
				if releaseOk, releaseErr := redisLock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
					l.Errorf("释放分布式锁失败: %v", releaseErr)
				}
			}()
			l.rebuildInboxCacheBestEffort(l.ctx, userID, inboxKey)
		}
		return nil, errorx.NewMsg("查询失败请稍后重试")
	}

	if len(ids) == 0 {
		return &content.FollowFeedRes{
			Items:      []*content.FollowFeedItem{},
			NextCursor: "",
			HasMore:    false,
		}, nil
	}

	statusPublished := int32(content.ContentStatus_PUBLISHED)
	visibilityPublic := int32(content.Visibility_PUBLIC)
	contentMap, err := l.contentRepo.BatchGetRecommendByIDs(statusPublished, visibilityPublic, ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注内容失败"))
	}

	contents := make([]*model.RanFeedContent, 0, len(ids))
	for _, id := range ids {
		if row, ok := contentMap[id]; ok && row != nil {
			contents = append(contents, row)
		}
	}
	if len(contents) == 0 {
		return &content.FollowFeedRes{
			Items:      []*content.FollowFeedItem{},
			NextCursor: "",
			HasMore:    false,
		}, nil
	}

	articleMap, videoMap, err := l.buildBriefMaps(contents)
	if err != nil {
		return nil, err
	}
	userMap, likedMap, likeCountMap, err := l.buildUserAndLikeMaps(contents, userID)
	if err != nil {
		return nil, err
	}

	items := l.buildFollowItems(contents, articleMap, videoMap, userMap, likedMap, likeCountMap)

	return &content.FollowFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
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

func (l *FollowFeedLogic) buildBriefMaps(contents []*model.RanFeedContent) (map[int64]*model.RanFeedArticle, map[int64]*model.RanFeedVideo, error) {
	articleIDs := make([]int64, 0)
	videoIDs := make([]int64, 0)
	for _, r := range contents {
		switch content.ContentType(r.ContentType) {
		case content.ContentType_ARTICLE:
			articleIDs = append(articleIDs, r.ID)
		case content.ContentType_VIDEO:
			videoIDs = append(videoIDs, r.ID)
		}
	}

	articleMap, err := l.articleRepo.BatchGetBriefByContentIDs(articleIDs)
	if err != nil {
		return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询文章摘要失败"))
	}

	videoMap, err := l.videoRepo.BatchGetBriefByContentIDs(videoIDs)
	if err != nil {
		return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询视频摘要失败"))
	}

	return articleMap, videoMap, nil
}

func (l *FollowFeedLogic) buildUserAndLikeMaps(contents []*model.RanFeedContent, userID int64) (map[int64]*userservice.UserInfo, map[int64]bool, map[int64]int64, error) {
	authorIDs := make([]int64, 0, len(contents))
	authorSeen := make(map[int64]struct{}, len(contents))
	likeInfos := make([]*likeservice.LikeInfo, 0, len(contents))

	for _, r := range contents {
		if _, ok := authorSeen[r.UserID]; !ok {
			authorSeen[r.UserID] = struct{}{}
			authorIDs = append(authorIDs, r.UserID)
		}

		switch content.ContentType(r.ContentType) {
		case content.ContentType_ARTICLE:
			likeInfos = append(likeInfos, &likeservice.LikeInfo{
				ContentId: r.ID,
				Scene:     interaction.Scene_ARTICLE,
			})
		case content.ContentType_VIDEO:
			likeInfos = append(likeInfos, &likeservice.LikeInfo{
				ContentId: r.ID,
				Scene:     interaction.Scene_VIDEO,
			})
		}
	}

	var (
		userMap      map[int64]*userservice.UserInfo
		likedMap     map[int64]bool
		likeCountMap map[int64]int64
	)

	err := mr.Finish(
		func() error {
			if len(authorIDs) == 0 {
				userMap = map[int64]*userservice.UserInfo{}
				return nil
			}
			resp, err := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userservice.BatchGetUserReq{UserIds: authorIDs})
			if err != nil {
				return err
			}
			userMap = make(map[int64]*userservice.UserInfo, len(resp.Users))
			for _, u := range resp.Users {
				if u == nil {
					continue
				}
				userMap[u.UserId] = u
			}
			return nil
		},
		func() error {
			likedMap = map[int64]bool{}
			likeCountMap = map[int64]int64{}
			if len(likeInfos) == 0 {
				return nil
			}
			resp, err := l.svcCtx.LikesRpc.BatchQueryLikeInfo(l.ctx, &likeservice.BatchQueryLikeInfoReq{
				UserId:    userID,
				LikeInfos: likeInfos,
			})
			if err != nil {
				return err
			}
			for _, info := range resp.LikeInfos {
				if info == nil {
					continue
				}
				likeCountMap[info.ContentId] = info.LikeCount
				if info.IsLiked {
					likedMap[info.ContentId] = true
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return userMap, likedMap, likeCountMap, nil
}

func (l *FollowFeedLogic) buildFollowItems(contents []*model.RanFeedContent, articleMap map[int64]*model.RanFeedArticle, videoMap map[int64]*model.RanFeedVideo, userMap map[int64]*userservice.UserInfo, likedMap map[int64]bool, likeCountMap map[int64]int64) []*content.FollowFeedItem {
	items := make([]*content.FollowFeedItem, 0, len(contents))
	for _, r := range contents {
		title := ""
		coverURL := ""
		switch content.ContentType(r.ContentType) {
		case content.ContentType_ARTICLE:
			if a, ok := articleMap[r.ID]; ok && a != nil {
				title = a.Title
				coverURL = a.Cover
			}
		case content.ContentType_VIDEO:
			if v, ok := videoMap[r.ID]; ok && v != nil {
				title = v.Title
				coverURL = v.CoverURL
			}
		}

		authorName := ""
		authorAvatar := ""
		if u, ok := userMap[r.UserID]; ok && u != nil {
			authorName = u.Nickname
			authorAvatar = u.Avatar
		}

		publishedAt := int64(0)
		if r.PublishedAt != nil {
			publishedAt = r.PublishedAt.Unix()
		} else {
			publishedAt = time.Now().Unix()
		}

		items = append(items, &content.FollowFeedItem{
			ContentId:    r.ID,
			ContentType:  content.ContentType(r.ContentType),
			AuthorId:     r.UserID,
			AuthorName:   authorName,
			AuthorAvatar: authorAvatar,
			Title:        title,
			CoverUrl:     coverURL,
			PublishedAt:  publishedAt,
			IsLiked:      likedMap[r.ID],
			LikeCount:    likeCountMap[r.ID],
		})
	}
	return items
}
