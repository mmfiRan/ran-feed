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
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	userPublishFeedKeepN = 5000

	userPublishFeedRebuildLockTTLSeconds = 30
	userPublishFeedRebuildRetryTimes     = 3
	userPublishFeedRebuildRetryInterval  = 80 * time.Millisecond
)

type UserPublishFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewUserPublishFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishFeedLogic {
	return &UserPublishFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *UserPublishFeedLogic) UserPublishFeed(in *content.UserPublishFeedReq) (*content.UserPublishFeedRes, error) {
	if in == nil {
		return emptyUserPublishFeedRes(), nil
	}
	if in.AuthorId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	feedKey := buildUserPublishFeedKey(in.AuthorId)

	ids, nextCursor, hasMore, err := l.loadPageIDs(feedKey, in.AuthorId, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if ids == nil || len(ids) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	contents, err := l.loadContents(ids)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	return l.assembleResponse(contents, nextCursor, hasMore, in.ViewerId)
}

func emptyUserPublishFeedRes() *content.UserPublishFeedRes {
	return &content.UserPublishFeedRes{
		Items:      []*content.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

func (l *UserPublishFeedLogic) loadPageIDs(feedKey string, authorID int64, cursor string, pageSize int) ([]int64, string, bool, error) {
	// 优先走 Redis zset，拿到当前页 content_id 列表
	ids, nextCursor, hasMore, cacheExists, err := l.queryUserPublishIDs(feedKey, cursor, pageSize)
	if err != nil {
		return nil, "", false, err
	}
	if cacheExists {
		return ids, nextCursor, hasMore, nil
	}

	// 缓存不存在时回源 DB，并用分布式锁避免并发重建击穿。
	lockKey := buildUserPublishFeedRebuildLockKey(authorID)
	rebuildLock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
	rebuildLock.SetExpire(userPublishFeedRebuildLockTTLSeconds)
	locked, lockErr := rebuildLock.AcquireCtx(l.ctx)
	if lockErr != nil {
		return nil, "", false, errorx.Wrap(l.ctx, lockErr, errorx.NewMsg("查询失败请稍后重试"))
	}
	if locked {
		defer func() {
			if releaseOk, releaseErr := rebuildLock.ReleaseCtx(context.Background()); !releaseOk || releaseErr != nil {
				l.Errorf("释放分布式锁失败: %v", releaseErr)
			}
		}()

		// 抢到锁的请求负责全量构建该作者发布列表缓存
		allRows, qerr := l.queryUserPublishAllFromDB(authorID)
		if qerr != nil {
			return nil, "", false, qerr
		}
		if len(allRows) == 0 {
			return nil, "", false, nil
		}
		if uerr := l.updateUserPublishCache(feedKey, allRows); uerr != nil {
			l.Errorf("回填用户发布列表缓存失败:%v", uerr)
		}

		// 回源场景直接在内存中按 cursor 做一次分页，避免再走一遍 Redis
		pageRows := l.pageUserPublishRows(allRows, cursor, pageSize)
		if len(pageRows) > pageSize {
			hasMore = true
			nextCursor = strconv.FormatInt(pageRows[pageSize-1].ID, 10)
			pageRows = pageRows[:pageSize]
		} else {
			hasMore = false
			nextCursor = ""
		}

		res := make([]int64, 0, len(pageRows))
		for _, row := range pageRows {
			if row == nil || row.ID <= 0 {
				continue
			}
			res = append(res, row.ID)
		}
		return res, nextCursor, hasMore, nil
	}

	// 未抢到锁的请求短暂等待并重试读取缓存。
	for i := 0; i < userPublishFeedRebuildRetryTimes; i++ {
		time.Sleep(userPublishFeedRebuildRetryInterval)
		ids, nextCursor, hasMore, cacheExists, err = l.queryUserPublishIDs(feedKey, cursor, pageSize)
		if err != nil {
			return nil, "", false, err
		}
		if cacheExists {
			return ids, nextCursor, hasMore, nil
		}
	}
	return nil, "", false, errorx.NewMsg("查询失败请稍后重试")
}

func (l *UserPublishFeedLogic) loadContents(ids []int64) ([]*model.RanFeedContent, error) {
	contentMap, err := l.contentRepo.BatchGetPublishedByIDs(ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布内容失败"))
	}

	contents := make([]*model.RanFeedContent, 0, len(ids))
	// 按 ids 原顺序重建，保证返回顺序与缓存分页顺序一致。
	for _, id := range ids {
		if row, ok := contentMap[id]; ok && row != nil {
			contents = append(contents, row)
		}
	}
	return contents, nil
}

func (l *UserPublishFeedLogic) assembleResponse(contents []*model.RanFeedContent, nextCursor string, hasMore bool, viewerID *int64) (*content.UserPublishFeedRes, error) {
	return l.buildUserPublishResponse(contents, nextCursor, hasMore, viewerID)
}

func buildUserPublishFeedKey(authorID int64) string {
	return rediskey.GetRedisPrefixKey("feed:user:publish", strconv.FormatInt(authorID, 10))
}

func buildUserPublishFeedRebuildLockKey(authorID int64) string {
	return rediskey.GetRedisPrefixKey("feed:user:publish:lock", strconv.FormatInt(authorID, 10))
}

func (l *UserPublishFeedLogic) queryUserPublishIDs(feedKey, cursor string, pageSize int) ([]int64, string, bool, bool, error) {
	// Lua 返回: [keyExists, hasMore, nextCursor, id1, id2, ...]
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserPublishZSetScript,
		[]string{feedKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布列表失败"))
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, "", false, false, errorx.NewMsg("查询发布列表失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	cacheExists := existsVal == 1
	if !cacheExists {
		return nil, "", false, false, nil
	}

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
	return ids, nextCursor, hasMore, true, nil
}

func (l *UserPublishFeedLogic) queryUserPublishAllFromDB(authorID int64) ([]*model.RanFeedContent, error) {
	rows, err := l.contentRepo.ListPublishedByAuthor(authorID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布内容失败"))
	}
	return rows, nil
}

func (l *UserPublishFeedLogic) updateUserPublishCache(feedKey string, rows []*model.RanFeedContent) error {
	if len(rows) == 0 {
		return nil
	}
	args := make([]interface{}, 0, 1+len(rows)*2)
	args = append(args, strconv.FormatInt(int64(userPublishFeedKeepN), 10))
	// UpdateUserPublishZSetScript 参数约定: [keep_latest_n, member, score, ...]
	// 这里使用 content_id 作为 member 和 score，天然可做时间倒序游标分页。
	for _, r := range rows {
		if r == nil {
			continue
		}
		idStr := strconv.FormatInt(r.ID, 10)
		args = append(args, idStr, idStr)
	}
	_, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.UpdateUserPublishZSetScript,
		[]string{feedKey},
		args...,
	)
	return err
}

func (l *UserPublishFeedLogic) pageUserPublishRows(allRows []*model.RanFeedContent, cursor string, pageSize int) []*model.RanFeedContent {
	if len(allRows) == 0 {
		return allRows
	}

	cursorID := int64(0)
	if cursor != "" {
		v, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil && v > 0 {
			cursorID = v
		}
	}

	res := make([]*model.RanFeedContent, 0, pageSize+1)
	// 多取 1 条用于判断 hasMore，避免额外 count 查询。
	for _, r := range allRows {
		if r == nil {
			continue
		}
		if cursorID > 0 && r.ID >= cursorID {
			continue
		}
		res = append(res, r)
		if len(res) >= pageSize+1 {
			break
		}
	}
	return res
}

func (l *UserPublishFeedLogic) buildUserPublishResponse(contents []*model.RanFeedContent, nextCursor string, hasMore bool, viewerID *int64) (*content.UserPublishFeedRes, error) {
	// 先批量补齐内容摘要，再补齐作者/点赞信息，最后统一组装响应
	articleMap, videoMap, err := l.buildBriefMaps(contents)
	if err != nil {
		return nil, err
	}

	vid := int64(0)
	if viewerID != nil {
		vid = *viewerID
	}
	userMap, likedMap, likeCountMap, err := l.buildUserAndLikeMaps(contents, vid)
	if err != nil {
		return nil, err
	}

	items := l.buildItems(contents, articleMap, videoMap, userMap, likedMap, likeCountMap)
	return &content.UserPublishFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (l *UserPublishFeedLogic) buildBriefMaps(contents []*model.RanFeedContent) (map[int64]*model.RanFeedArticle, map[int64]*model.RanFeedVideo, error) {
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

func (l *UserPublishFeedLogic) buildUserAndLikeMaps(contents []*model.RanFeedContent, userID int64) (map[int64]*userservice.UserInfo, map[int64]bool, map[int64]int64, error) {
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

	// 用户信息与点赞信息并行查询
	err := mr.Finish(
		func() error {
			if len(authorIDs) == 0 {
				userMap = map[int64]*userservice.UserInfo{}
				return nil
			}
			resp, err := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userservice.BatchGetUserReq{
				UserIds: authorIDs,
			})
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

func (l *UserPublishFeedLogic) buildItems(contents []*model.RanFeedContent, articleMap map[int64]*model.RanFeedArticle, videoMap map[int64]*model.RanFeedVideo, userMap map[int64]*userservice.UserInfo, likedMap map[int64]bool, likeCountMap map[int64]int64) []*content.ContentItem {
	items := make([]*content.ContentItem, 0, len(contents))
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
			// 兜底避免返回 0 时间戳影响前端展示
			publishedAt = time.Now().Unix()
		}

		item := &content.ContentItem{
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
		}
		items = append(items, item)
	}
	return items
}
