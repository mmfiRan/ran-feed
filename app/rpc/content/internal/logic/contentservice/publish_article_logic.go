package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishArticleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepository repositories.ContentRepository
	articleRepository repositories.ArticleRepository
}

func NewPublishArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishArticleLogic {
	return &PublishArticleLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		Logger:            logx.WithContext(ctx),
		contentRepository: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepository: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *PublishArticleLogic) PublishArticle(in *content.ArticlePublishReq) (*content.ArticlePublishRes, error) {
	if err := validateArticlePublish(in.Title, in.Cover, in.Content); err != nil {
		return nil, err
	}
	visibility, err := resolveWriteVisibility(writeModePublish, in.Visibility, 0)
	if err != nil {
		return nil, err
	}

	var contentId int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		articleRepo := l.articleRepository.WithTx(tx)

		contentDO := buildContentDO(in.UserId, content.ContentType_CONTENT_TYPE_ARTICLE,
			content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW, visibility)
		contentId = contentDO.ID
		if err := contentRepo.CreateContent(contentDO); err != nil {
			return err
		}
		articleDO := &do.ArticleDO{
			ID:          snowflake.GenID(),
			ContentID:   contentId,
			Title:       in.Title,
			Description: in.Description,
			Cover:       in.Cover,
			Content:     in.Content,
		}
		return articleRepo.CreateArticle(articleDO)
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布文章失败"))
	}

	return &content.ArticlePublishRes{
		ContentId: contentId,
	}, nil
}
