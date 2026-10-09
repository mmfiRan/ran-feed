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

type PublishArticleLogic struct {
	ctx context.Context
	logx.Logger
	contentRepository repositories.ContentRepository
	articleRepository repositories.ArticleRepository
}

func NewPublishArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishArticleLogic {
	return &PublishArticleLogic{
		ctx:               ctx,
		Logger:            logx.WithContext(ctx),
		contentRepository: svcCtx.ContentRepository,
		articleRepository: svcCtx.ArticleRepository,
	}
}

func (l *PublishArticleLogic) PublishArticle(in *content.ArticlePublishReq) (*content.ArticlePublishRes, error) {
	if err := validateArticlePublish(in.Title, in.Cover, in.Content); err != nil {
		return nil, err
	}
	visibility, err := resolveWriteVisibility(writeModePublish, in.Visibility, contentEnum.VisibilityUnknown)
	if err != nil {
		return nil, err
	}

	var contentId int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		articleRepo := l.articleRepository.WithTx(tx)

		contentModel := buildContentModel(in.UserId, contentEnum.ContentTypeArticle,
			contentEnum.ContentStatusPendingReview, visibility)
		contentId = contentModel.ID
		if err := contentRepo.CreateContent(l.ctx, contentModel); err != nil {
			return err
		}
		articleModel := &model.RanFeedArticle{
			ID:          snowflake.GenID(),
			ContentID:   contentId,
			Title:       in.Title,
			Description: in.Description,
			Cover:       in.Cover,
			Content:     in.Content,
		}
		return articleRepo.CreateArticle(l.ctx, articleModel)
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布文章失败"))
	}

	return &content.ArticlePublishRes{
		ContentId: contentId,
	}, nil
}
