package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type GetContentDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

type contentCounts struct {
	LikeCount     int64
	FavoriteCount int64
	CommentCount  int64
}

func NewGetContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetContentDetailLogic {
	return &GetContentDetailLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetContentDetailLogic) GetContentDetail(in *content.GetContentDetailReq) (*content.GetContentDetailRes, error) {
	if in == nil || in.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	contentRow, err := l.contentRepo.GetDetailByID(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容详情失败"))
	}
	if contentRow == nil || contentRow.Status != int32(content.ContentStatus_PUBLISHED) {
		return nil, errorx.NewMsg("内容不存在")
	}

	viewerID := in.GetViewerId()
	if contentRow.Visibility == int32(content.Visibility_PRIVATE) && viewerID != contentRow.UserID {
		return nil, errorx.NewMsg("内容不存在")
	}

	detail, err := l.buildDetail(contentRow, viewerID)
	if err != nil {
		return nil, err
	}

	return &content.GetContentDetailRes{
		Detail: detail,
	}, nil
}

func (l *GetContentDetailLogic) buildDetail(contentRow *model.RanFeedContent, viewerID int64) (*content.ContentDetail, error) {
	detail := &content.ContentDetail{
		ContentId:   contentRow.ID,
		ContentType: content.ContentType(contentRow.ContentType),
		AuthorId:    contentRow.UserID,
	}
	if contentRow.PublishedAt != nil {
		detail.PublishedAt = contentRow.PublishedAt.Unix()
	}

	scene := interaction.Scene_SCENE_UNKNOWN
	if err := l.fillContentFields(detail, contentRow.ID, content.ContentType(contentRow.ContentType)); err != nil {
		return nil, err
	}
	switch detail.ContentType {
	case content.ContentType_ARTICLE:
		scene = interaction.Scene_ARTICLE
	case content.ContentType_VIDEO:
		scene = interaction.Scene_VIDEO
	default:
		return nil, errorx.NewMsg("内容类型错误")
	}

	author, likeInfo, favoriteInfo, followInfo, counts, err := l.loadExtraInfo(contentRow.UserID, contentRow.ID, viewerID, scene)
	if err != nil {
		return nil, err
	}
	if author != nil {
		detail.AuthorName = author.Nickname
		detail.AuthorAvatar = author.Avatar
	}
	if likeInfo != nil {
		detail.IsLiked = likeInfo.IsLiked
	}
	if favoriteInfo != nil {
		detail.IsFavorited = favoriteInfo.IsFavorited
	}
	if followInfo != nil {
		detail.IsFollowingAuthor = followInfo.IsFollowing
	}
	if counts != nil {
		detail.LikeCount = counts.LikeCount
		detail.FavoriteCount = counts.FavoriteCount
		detail.CommentCount = counts.CommentCount
	}

	return detail, nil
}

func (l *GetContentDetailLogic) fillContentFields(detail *content.ContentDetail, contentID int64, contentType content.ContentType) error {
	switch contentType {
	case content.ContentType_ARTICLE:
		articleRow, err := l.articleRepo.GetByContentID(contentID)
		if err != nil {
			return errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容详情失败"))
		}
		if articleRow == nil {
			return errorx.NewMsg("内容不存在")
		}

		detail.Title = articleRow.Title
		if articleRow.Description != nil {
			detail.Description = *articleRow.Description
		}
		detail.CoverUrl = articleRow.Cover
		detail.ArticleContent = articleRow.Content
		return nil
	case content.ContentType_VIDEO:
		videoRow, err := l.videoRepo.GetByContentID(contentID)
		if err != nil {
			return errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容详情失败"))
		}
		if videoRow == nil {
			return errorx.NewMsg("内容不存在")
		}

		detail.Title = videoRow.Title
		detail.CoverUrl = videoRow.CoverURL
		detail.VideoUrl = videoRow.OriginURL
		detail.VideoDuration = videoRow.Duration
		return nil
	default:
		return errorx.NewMsg("内容类型错误")
	}
}

func (l *GetContentDetailLogic) loadExtraInfo(authorID, contentID, viewerID int64, scene interaction.Scene) (*userservice.UserInfo, *likeservice.QueryLikeInfoRes, *favoriteservice.QueryFavoriteInfoRes, *followservice.GetFollowSummaryRes, *contentCounts, error) {
	var (
		author       *userservice.UserInfo
		likeInfo     *likeservice.QueryLikeInfoRes
		favoriteInfo *favoriteservice.QueryFavoriteInfoRes
		followInfo   *followservice.GetFollowSummaryRes
		counts       = &contentCounts{}
	)

	err := mr.Finish(
		func() error {
			resp, err := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userservice.BatchGetUserReq{
				UserIds: []int64{authorID},
			})
			if err != nil {
				return err
			}
			if len(resp.Users) > 0 {
				author = resp.Users[0]
			}
			return nil
		},
		func() error {
			resp, err := l.svcCtx.LikesRpc.BatchQueryLikeInfo(l.ctx, &likeservice.BatchQueryLikeInfoReq{
				UserId: viewerID,
				LikeInfos: []*likeservice.LikeInfo{
					{
						ContentId: contentID,
						Scene:     scene,
					},
				},
			})
			if err != nil {
				return err
			}
			if len(resp.LikeInfos) > 0 {
				likeInfo = resp.LikeInfos[0]
			}
			return nil
		},
		func() error {
			resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteInfo(l.ctx, &favoriteservice.QueryFavoriteInfoReq{
				UserId:    viewerID,
				ContentId: contentID,
				Scene:     scene,
			})
			if err != nil {
				return err
			}
			favoriteInfo = resp
			return nil
		},
		func() error {
			viewerID := viewerID
			resp, err := l.svcCtx.FollowRpc.GetFollowSummary(l.ctx, &followservice.GetFollowSummaryReq{
				UserId:   authorID,
				ViewerId: &viewerID,
			})
			if err != nil {
				return err
			}
			followInfo = resp
			return nil
		},
		func() error {
			resp, err := l.svcCtx.CountRpc.BatchGetCount(l.ctx, &count.BatchGetCountReq{
				Keys: []*count.CountKey{
					{BizType: count.BizType_LIKE, TargetType: count.TargetType_CONTENT, TargetId: contentID},
					{BizType: count.BizType_FAVORITE, TargetType: count.TargetType_CONTENT, TargetId: contentID},
					{BizType: count.BizType_COMMENT, TargetType: count.TargetType_CONTENT, TargetId: contentID},
				},
			})
			if err != nil {
				return err
			}
			for _, item := range resp.Items {
				if item == nil || item.Key == nil {
					continue
				}
				switch item.Key.BizType {
				case count.BizType_LIKE:
					counts.LikeCount = item.Value
				case count.BizType_FAVORITE:
					counts.FavoriteCount = item.Value
				case count.BizType_COMMENT:
					counts.CommentCount = item.Value
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, nil, nil, nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容详情失败"))
	}

	return author, likeInfo, favoriteInfo, followInfo, counts, nil
}
