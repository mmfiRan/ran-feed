package contentservicelogic

import (
	"context"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

const userPublishFeedKeepN = 5000

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
	var contentId int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		articleRepo := l.articleRepository.WithTx(tx)

		contentId = snowflake.GenID()
		now := time.Now()
		contentDO := &do.ContentDO{
			ID:          contentId,
			UserID:      in.UserId,
			ContentType: int32(content.ContentType_ARTICLE),
			Status:      int32(content.ContentStatus_PUBLISHED),
			Visibility:  int32(in.Visibility),
			PublishedAt: &now,
			CreatedBy:   in.UserId,
			UpdatedBy:   in.UserId,
		}
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

		if err := articleRepo.CreateArticle(articleDO); err != nil {
			return err
		}

		feedKey := rediskey.BuildUserPublishFeedKey(in.UserId)
		contentIDStr := strconv.FormatInt(contentId, 10)
		_, cacheErr := l.svcCtx.Redis.EvalCtx(
			l.ctx,
			luautils.UpdateUserPublishZSetScript,
			[]string{feedKey},
			strconv.FormatInt(int64(userPublishFeedKeepN), 10),
			contentIDStr, contentIDStr,
		)
		if cacheErr != nil {
			return cacheErr
		}
		if shouldSeedHotIncrement(in.Visibility) {
			if err := writePublishHotSeed(l.ctx, l.svcCtx, contentId); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布文章失败"))
	}

	return &content.ArticlePublishRes{
		ContentId: contentId,
	}, nil
}
