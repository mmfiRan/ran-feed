package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

// contentDetailResolver feed 读路径共享件 统一二级缓存内容详情读取与拼装
type contentDetailResolver struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func newContentDetailResolver(ctx context.Context, svcCtx *svc.ServiceContext) *contentDetailResolver {
	return &contentDetailResolver{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

// resolveDetails 走 L2 二级缓存按 ids 顺序拿内容详情
// publicOnly 为真时只保留 PUBLIC 已删/未发布的 id 自动剔除
func (r *contentDetailResolver) resolveDetails(ids []int64, publicOnly bool) ([]*do.ContentDetailDO, error) {
	detailMap, err := contentcache.BatchGet(r.ctx, r.svcCtx.Redis, r.svcCtx.Config.ContentCache, ids, r.loadDetails)
	if err != nil {
		return nil, err
	}

	details := make([]*do.ContentDetailDO, 0, len(ids))
	for _, id := range ids {
		d, ok := detailMap[id]
		if !ok || d == nil {
			continue
		}
		if publicOnly && d.Visibility != int32(content.Visibility_PUBLIC) {
			continue
		}
		details = append(details, d)
	}
	return details, nil
}

// assembleItems 内容的完整消息
func (r *contentDetailResolver) assembleItems(ids []int64, viewerID int64, publicOnly bool) ([]*content.ContentItem, error) {
	details, err := r.resolveDetails(ids, publicOnly)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return []*content.ContentItem{}, nil
	}

	userMap, likedMap, likeCountMap, err := r.loadAuthorsAndLikes(details, viewerID)
	if err != nil {
		return nil, err
	}
	return buildContentItems(details, userMap, likedMap, likeCountMap), nil
}

// loadDetails L2 miss 回源 content 行加 article/video brief 拼内容本征详情
func (r *contentDetailResolver) loadDetails(missIDs []int64) (map[int64]*do.ContentDetailDO, error) {
	contentMap, err := r.contentRepo.BatchGetPublishedByIDs(missIDs)
	if err != nil {
		return nil, errorx.Wrap(r.ctx, err, errorx.NewMsg("查询内容失败"))
	}
	if len(contentMap) == 0 {
		return map[int64]*do.ContentDetailDO{}, nil
	}

	articleIDs := make([]int64, 0)
	videoIDs := make([]int64, 0)
	for _, row := range contentMap {
		switch content.ContentType(row.ContentType) {
		case content.ContentType_ARTICLE:
			articleIDs = append(articleIDs, row.ID)
		case content.ContentType_VIDEO:
			videoIDs = append(videoIDs, row.ID)
		}
	}

	articleMap, err := r.articleRepo.BatchGetBriefByContentIDs(articleIDs)
	if err != nil {
		return nil, errorx.Wrap(r.ctx, err, errorx.NewMsg("查询文章摘要失败"))
	}
	videoMap, err := r.videoRepo.BatchGetBriefByContentIDs(videoIDs)
	if err != nil {
		return nil, errorx.Wrap(r.ctx, err, errorx.NewMsg("查询视频摘要失败"))
	}

	res := make(map[int64]*do.ContentDetailDO, len(contentMap))
	for id, row := range contentMap {
		title := ""
		coverURL := ""
		switch content.ContentType(row.ContentType) {
		case content.ContentType_ARTICLE:
			if a, ok := articleMap[row.ID]; ok && a != nil {
				title = a.Title
				coverURL = a.Cover
			}
		case content.ContentType_VIDEO:
			if v, ok := videoMap[row.ID]; ok && v != nil {
				title = v.Title
				coverURL = v.CoverURL
			}
		}
		publishedAt := int64(0)
		if row.PublishedAt != nil {
			publishedAt = row.PublishedAt.Unix()
		}
		res[id] = &do.ContentDetailDO{
			ContentID:   row.ID,
			ContentType: row.ContentType,
			AuthorID:    row.UserID,
			Title:       title,
			CoverURL:    coverURL,
			PublishedAt: publishedAt,
			Visibility:  row.Visibility,
		}
	}
	return res, nil
}

// loadAuthorsAndLikes 并行查作者信息走 user usercache 与点赞信息观察者相关不缓存
func (r *contentDetailResolver) loadAuthorsAndLikes(details []*do.ContentDetailDO, viewerID int64) (map[int64]*userservice.UserInfo, map[int64]bool, map[int64]int64, error) {
	authorIDs := make([]int64, 0, len(details))
	authorSeen := make(map[int64]struct{}, len(details))
	likeInfos := make([]*likeservice.LikeInfo, 0, len(details))

	for _, d := range details {
		if _, ok := authorSeen[d.AuthorID]; !ok {
			authorSeen[d.AuthorID] = struct{}{}
			authorIDs = append(authorIDs, d.AuthorID)
		}
		switch content.ContentType(d.ContentType) {
		case content.ContentType_ARTICLE:
			likeInfos = append(likeInfos, &likeservice.LikeInfo{ContentId: d.ContentID, Scene: interaction.Scene_ARTICLE})
		case content.ContentType_VIDEO:
			likeInfos = append(likeInfos, &likeservice.LikeInfo{ContentId: d.ContentID, Scene: interaction.Scene_VIDEO})
		}
	}

	var (
		userMap      map[int64]*userservice.UserInfo
		likedMap     map[int64]bool
		likeCountMap map[int64]int64
	)

	err := mr.Finish(
		func() error {
			userMap = map[int64]*userservice.UserInfo{}
			if len(authorIDs) == 0 {
				return nil
			}
			resp, err := r.svcCtx.UserRpc.BatchGetUser(r.ctx, &userservice.BatchGetUserReq{UserIds: authorIDs})
			if err != nil {
				return err
			}
			if resp == nil {
				return nil
			}
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
			resp, err := r.svcCtx.LikesRpc.BatchQueryLikeInfo(r.ctx, &likeservice.BatchQueryLikeInfoReq{
				UserId:    viewerID,
				LikeInfos: likeInfos,
			})
			if err != nil {
				return err
			}
			if resp == nil {
				return nil
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

// buildContentItems 按 details 顺序把 L2 详情加作者加点赞组装成 ContentItem
func buildContentItems(details []*do.ContentDetailDO, userMap map[int64]*userservice.UserInfo, likedMap map[int64]bool, likeCountMap map[int64]int64) []*content.ContentItem {
	items := make([]*content.ContentItem, 0, len(details))
	for _, d := range details {
		authorName := ""
		authorAvatar := ""
		if u, ok := userMap[d.AuthorID]; ok && u != nil {
			authorName = u.Nickname
			authorAvatar = u.Avatar
		}
		items = append(items, &content.ContentItem{
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
