package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/logichelper"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GetContentDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
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
	if contentRow == nil || contentRow.Status != int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED) {
		return nil, errorx.NewMsg("内容不存在")
	}

	viewerID := in.GetViewerId()
	if contentRow.Visibility == int32(content.Visibility_VISIBILITY_PRIVATE) && viewerID != contentRow.UserID {
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
	contentType := content.ContentType(contentRow.ContentType)
	detail := &content.ContentDetail{
		ContentId:   contentRow.ID,
		ContentType: logichelper.ContentTypeValue(contentRow.ContentType),
		AuthorId:    contentRow.UserID,
	}
	if contentRow.PublishedAt != nil {
		detail.PublishedAt = timestamppb.New(*contentRow.PublishedAt)
	}

	var scene interaction.Scene
	if err := l.fillContentFields(detail, contentRow.ID, contentType); err != nil {
		return nil, err
	}
	switch contentType {
	case content.ContentType_CONTENT_TYPE_ARTICLE:
		scene = interaction.Scene_ARTICLE
	case content.ContentType_CONTENT_TYPE_VIDEO:
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
		detail.LikeCount = counts.GetLikeCount()
		detail.FavoriteCount = counts.GetFavoriteCount()
		detail.CommentCount = counts.GetCommentCount()
	}

	return detail, nil
}

func (l *GetContentDetailLogic) fillContentFields(detail *content.ContentDetail, contentID int64, contentType content.ContentType) error {
	switch contentType {
	case content.ContentType_CONTENT_TYPE_ARTICLE:
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
	case content.ContentType_CONTENT_TYPE_VIDEO:
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

func (l *GetContentDetailLogic) loadExtraInfo(authorID, contentID, viewerID int64, scene interaction.Scene) (*user.UserInfo, *likeservice.QueryLikeInfoRes, *favoriteservice.QueryFavoriteInfoRes, *followservice.GetFollowSummaryRes, *count.ContentCountsItem, error) {
	var (
		author       *user.UserInfo
		likeInfo     *likeservice.QueryLikeInfoRes
		favoriteInfo *favoriteservice.QueryFavoriteInfoRes
		followInfo   *followservice.GetFollowSummaryRes
		counts       *count.ContentCountsItem
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
			resp, err := l.svcCtx.CountRpc.BatchGetContentCounts(l.ctx, &count.BatchGetContentCountsReq{
				ContentIds: []int64{contentID},
			})
			if err != nil {
				return err
			}
			if items := resp.GetItems(); len(items) > 0 {
				counts = items[0]
			}
			return nil
		},
	)
	if err != nil {
		return nil, nil, nil, nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容详情失败"))
	}

	return author, likeInfo, favoriteInfo, followInfo, counts, nil
}
