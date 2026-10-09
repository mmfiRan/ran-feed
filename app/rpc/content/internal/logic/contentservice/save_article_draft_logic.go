package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveArticleDraftLogic struct {
	ctx context.Context
	logx.Logger
	contentRepository repositories.ContentRepository
	articleRepository repositories.ArticleRepository
}

func NewSaveArticleDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveArticleDraftLogic {
	return &SaveArticleDraftLogic{
		ctx:               ctx,
		Logger:            logx.WithContext(ctx),
		contentRepository: svcCtx.ContentRepository,
		articleRepository: svcCtx.ArticleRepository,
	}
}

func (l *SaveArticleDraftLogic) SaveArticleDraft(in *content.SaveArticleDraftReq) (*content.SaveArticleDraftRes, error) {
	// 有 content_id 走更新 否则新建草稿
	if in.ContentId != nil && *in.ContentId > 0 {
		contentID, err := l.updateDraft(in, *in.ContentId)
		if err != nil {
			return nil, err
		}
		return &content.SaveArticleDraftRes{ContentId: contentID}, nil
	}

	contentID, err := l.createDraft(in)
	if err != nil {
		return nil, err
	}
	return &content.SaveArticleDraftRes{
		ContentId: contentID,
	}, nil
}

func (l *SaveArticleDraftLogic) createDraft(in *content.SaveArticleDraftReq) (int64, error) {
	visibility, err := resolveWriteVisibility(writeModeDraft, in.Visibility, contentEnum.VisibilityPublic)
	if err != nil {
		return 0, err
	}

	var contentID int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		articleRepo := l.articleRepository.WithTx(tx)

		contentModel := buildContentModel(in.UserId, contentEnum.ContentTypeArticle,
			contentEnum.ContentStatusDraft, visibility)
		contentID = contentModel.ID
		if err := contentRepo.CreateContent(l.ctx, contentModel); err != nil {
			return err
		}
		articleModel := &model.RanFeedArticle{
			ID:          snowflake.GenID(),
			ContentID:   contentID,
			Title:       in.Title,
			Description: in.Description,
			Cover:       in.Cover,
			Content:     in.Content,
		}
		return articleRepo.CreateArticle(l.ctx, articleModel)
	}); err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存草稿失败"))
	}
	return contentID, nil
}

func (l *SaveArticleDraftLogic) updateDraft(in *content.SaveArticleDraftReq, contentID int64) (int64, error) {
	row, err := l.contentRepository.GetOwnedByID(l.ctx, contentID, in.UserId)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存草稿失败"))
	}
	if row == nil {
		return 0, errorx.NewMsg("内容不存在或无权限")
	}
	if contentEnum.ContentTypeEnum(row.ContentType) != contentEnum.ContentTypeArticle {
		return 0, errorx.NewMsg("内容类型不匹配")
	}
	if !isEditableStatus(contentEnum.ContentStatusEnum(row.Status)) {
		return 0, errorx.NewMsg("该内容当前状态不可编辑")
	}

	visibility, err := resolveWriteVisibility(writeModeDraft, in.Visibility, contentEnum.VisibilityEnum(row.Visibility))
	if err != nil {
		return 0, err
	}

	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		articleRepo := l.articleRepository.WithTx(tx)

		if err := contentRepo.UpdateDraftMeta(l.ctx, contentID, visibility, in.UserId); err != nil {
			return err
		}
		articleModel := &model.RanFeedArticle{
			ContentID:   contentID,
			Title:       in.Title,
			Description: in.Description,
			Cover:       in.Cover,
			Content:     in.Content,
		}
		return articleRepo.UpdateByContentID(l.ctx, articleModel)
	}); err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存草稿失败"))
	}
	return contentID, nil
}

// isEditableStatus 仅草稿与被拒内容可编辑
func isEditableStatus(status contentEnum.ContentStatusEnum) bool {
	return status == contentEnum.ContentStatusDraft || status == contentEnum.ContentStatusRejected
}
