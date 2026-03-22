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
)

type CacheResult int

const (
	CacheHit CacheResult = iota
	CacheMiss
	CacheError
)

type hotFeedResult struct {
	ids                []int64
	nextCursor         int64
	hasMore            bool
	resolvedSnapshotID string
}

type RecommendFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RecommendFeedLogic) RecommendFeed(in *content.RecommendFeedReq) (*content.RecommendFeedRes, error) {
	pageSize := int(in.PageSize)

	// 解析快照id和快照key
	preferredKey, preferredSnapshotID := l.resolveSnapshotKey(in.SnapshotId)

	// 从 Redis取id
	res, err := l.queryHotIDsByCursor(preferredKey, preferredSnapshotID, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if len(res.ids) == 0 {
		return &content.RecommendFeedRes{
			Items:      []*content.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: res.resolvedSnapshotID,
		}, nil
	}

	statusPublished := int32(content.ContentStatus_PUBLISHED)
	visibilityPublic := int32(content.Visibility_PUBLIC)
	contentMap, err := l.contentRepo.BatchGetRecommendByIDs(statusPublished, visibilityPublic, res.ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询热榜内容失败"))
	}

	contents := make([]*model.RanFeedContent, 0, len(res.ids))
	for _, id := range res.ids {
		if row, ok := contentMap[id]; ok && row != nil {
			contents = append(contents, row)
		}
	}
	if len(contents) == 0 {
		return &content.RecommendFeedRes{
			Items:      nil,
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: res.resolvedSnapshotID,
		}, nil
	}

	articleMap, videoMap, err := l.buildBriefMaps(contents)
	if err != nil {
		return nil, err
	}

	userID := int64(0)
	if in.UserId != nil {
		userID = *in.UserId
	}
	userMap, likedMap, likeCountMap, err := l.buildUserAndLikeMaps(contents, userID)
	if err != nil {
		return nil, err
	}

	items := l.buildItems(contents, articleMap, videoMap, userMap, likedMap, likeCountMap)

	return &content.RecommendFeedRes{
		Items:      items,
		NextCursor: res.nextCursor,
		HasMore:    res.hasMore,
		SnapshotId: res.resolvedSnapshotID,
	}, nil
}

func (l *RecommendFeedLogic) resolveSnapshotKey(reqSnapshotID *string) (string, string) {
	if reqSnapshotID == nil || *reqSnapshotID == "" {
		return "", ""
	}
	return rediskey.BuildHotFeedSnapshotKey(*reqSnapshotID), *reqSnapshotID
}

func (l *RecommendFeedLogic) queryHotIDsByCursor(preferredKey, preferredSnapshotID string, cursorID string, pageSize int) (*hotFeedResult, error) {
	res, cacheResult := l.queryFromRedis(preferredKey, preferredSnapshotID, cursorID, pageSize)
	if cacheResult == CacheHit {
		return res, nil
	}
	return nil, errorx.NewMsg("热榜缓存不存在")
}

func (l *RecommendFeedLogic) queryFromRedis(preferredKey, preferredSnapshotID, cursor string, pageSize int) (*hotFeedResult, CacheResult) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryHotFeedZSetScript,
		[]string{
			preferredKey,
			rediskey.RedisFeedHotGlobalLatestKey,
			rediskey.RedisFeedHotGlobalSnapshotPrefix,
			rediskey.RedisFeedHotGlobalKey,
		},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
		preferredSnapshotID,
	)
	if err != nil {
		l.Errorf("Lua脚本执行失败: %v", err)
		return nil, CacheError
	}

	parsed, exists, parseErr := parseHotFeedLuaResult(res)
	if parseErr != nil {
		l.Errorf("解析Lua返回值失败: %v", parseErr)
		return nil, CacheError
	}
	if !exists {
		return nil, CacheMiss
	}
	return parsed, CacheHit
}

func parseHotFeedLuaResult(res any) (*hotFeedResult, bool, error) {
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 4 {
		return nil, false, errorx.NewMsg("查询热榜索引失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	exists := existsVal == 1

	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1

	nextCursor := int64(0)
	if hasMore {
		nextStr, _ := luaReplyString(arr[2])
		if nextStr != "" {
			v, parseErr := strconv.ParseInt(nextStr, 10, 64)
			if parseErr != nil {
				return nil, false, errorx.NewMsg("查询热榜索引失败")
			}
			nextCursor = v
		}
	}

	resolvedSnapshotID, _ := luaReplyString(arr[3])

	ids := make([]int64, 0, len(arr)-4)
	for i := 4; i < len(arr); i++ {
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

	return &hotFeedResult{
		ids:                ids,
		nextCursor:         nextCursor,
		hasMore:            hasMore,
		resolvedSnapshotID: resolvedSnapshotID,
	}, exists, nil
}

func (l *RecommendFeedLogic) buildBriefMaps(contents []*model.RanFeedContent) (map[int64]*model.RanFeedArticle, map[int64]*model.RanFeedVideo, error) {
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

func (l *RecommendFeedLogic) buildUserAndLikeMaps(contents []*model.RanFeedContent, userID int64) (map[int64]*userservice.UserInfo, map[int64]bool, map[int64]int64, error) {
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
			likeInfos = append(likeInfos, &likeservice.LikeInfo{ContentId: r.ID, Scene: interaction.Scene_ARTICLE})
		case content.ContentType_VIDEO:
			likeInfos = append(likeInfos, &likeservice.LikeInfo{ContentId: r.ID, Scene: interaction.Scene_VIDEO})
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

func (l *RecommendFeedLogic) buildItems(contents []*model.RanFeedContent, articleMap map[int64]*model.RanFeedArticle, videoMap map[int64]*model.RanFeedVideo, userMap map[int64]*userservice.UserInfo, likedMap map[int64]bool, likeCountMap map[int64]int64) []*content.ContentItem {
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
			// 兜底时间
			publishedAt = time.Now().Unix()
		}

		items = append(items, &content.ContentItem{
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
