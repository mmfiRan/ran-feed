package admincontentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetContentDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewAdminGetContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetContentDetailLogic {
	return &AdminGetContentDetailLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *AdminGetContentDetailLogic) AdminGetContentDetail(in *content.AdminGetContentDetailReq) (*content.AdminGetContentDetailRes, error) {
	if in == nil || in.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	// 管理端取任意状态含非公开 无 viewer 门槛
	row, err := l.contentRepo.AdminGetByID(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容详情失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("内容不存在")
	}

	detail := &content.AdminContentDetail{
		ContentId:     row.ID,
		ContentType:   content.ContentType(row.ContentType),
		Status:        content.ContentStatus(row.Status),
		Visibility:    content.Visibility(row.Visibility),
		AuthorId:      row.UserID,
		LikeCount:     row.LikeCount,
		FavoriteCount: row.FavoriteCount,
		CommentCount:  row.CommentCount,
		CreatedAt:     row.CreatedAt.UnixMilli(),
		UpdatedAt:     row.UpdatedAt.UnixMilli(),
	}
	if row.PublishedAt != nil {
		detail.PublishedAt = row.PublishedAt.UnixMilli()
	}

	if err = l.fillContentFields(detail, row); err != nil {
		return nil, err
	}

	return &content.AdminGetContentDetailRes{Detail: detail}, nil
}

// fillContentFields 按类型回源 article/video 填标题/正文/封面等本征字段
func (l *AdminGetContentDetailLogic) fillContentFields(detail *content.AdminContentDetail, row *model.RanFeedContent) error {
	switch content.ContentType(row.ContentType) {
	case content.ContentType_ARTICLE:
		articleRow, err := l.articleRepo.GetByContentID(row.ID)
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
		videoRow, err := l.videoRepo.GetByContentID(row.ID)
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
