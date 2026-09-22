package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepository repositories.ContentRepository
	articleRepository repositories.ArticleRepository
	videoRepository   repositories.VideoRepository
}

func NewSubmitContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitContentLogic {
	return &SubmitContentLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		Logger:            logx.WithContext(ctx),
		contentRepository: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepository: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepository:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *SubmitContentLogic) SubmitContent(in *content.SubmitContentReq) (*content.SubmitContentRes, error) {
	row, err := l.contentRepository.GetOwnedByID(in.ContentId, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("提交失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("内容不存在或无权限")
	}
	if !isEditableStatus(row.Status) {
		return nil, errorx.NewMsg("该内容当前状态不可提交")
	}

	// 发布前完整性校验 草稿放宽 提交才强制
	if err := l.checkComplete(in.ContentId, content.ContentType(row.ContentType)); err != nil {
		return nil, err
	}

	affected, err := l.contentRepository.SubmitOwned(
		in.ContentId,
		in.UserId,
		[]int32{
			int32(content.ContentStatus_CONTENT_STATUS_DRAFT),
			int32(content.ContentStatus_CONTENT_STATUS_REJECTED),
		},
		int32(content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW),
		in.UserId,
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("提交失败"))
	}
	if affected == 0 {
		return nil, errorx.NewMsg("内容状态已变更 请刷新后重试")
	}

	return &content.SubmitContentRes{ContentId: in.ContentId}, nil
}

func (l *SubmitContentLogic) checkComplete(contentID int64, contentType content.ContentType) error {
	switch contentType {
	case content.ContentType_CONTENT_TYPE_ARTICLE:
		article, err := l.articleRepository.GetByContentID(contentID)
		if err != nil {
			return errorx.Wrap(l.ctx, err, errorx.NewMsg("提交失败"))
		}
		if article == nil || article.Title == "" || article.Cover == "" || article.Content == "" {
			return errorx.NewMsg("标题 封面 正文不能为空")
		}
		return nil
	case content.ContentType_CONTENT_TYPE_VIDEO:
		video, err := l.videoRepository.GetByContentID(contentID)
		if err != nil {
			return errorx.Wrap(l.ctx, err, errorx.NewMsg("提交失败"))
		}
		if video == nil || video.Title == "" || video.CoverURL == "" || video.OriginURL == "" || video.Duration <= 0 {
			return errorx.NewMsg("标题 封面 视频 时长不能为空")
		}
		return nil
	default:
		return errorx.NewMsg("内容类型错误")
	}
}
