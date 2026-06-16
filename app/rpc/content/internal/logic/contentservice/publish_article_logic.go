package contentservicelogic

import (
	"context"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
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
	now := time.Now()
	var contentId int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		articleRepo := l.articleRepository.WithTx(tx)

		contentId = snowflake.GenID()
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
		return articleRepo.CreateArticle(articleDO)
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布文章失败"))
	}

	l.afterPublish(contentId, in.UserId, now.UnixMilli(), in.Visibility)

	return &content.ArticlePublishRes{
		ContentId: contentId,
	}, nil
}

func (l *PublishArticleLogic) afterPublish(contentId, userID, publishedAtMillis int64, visibility content.Visibility) {
	feedKey := rediskey.BuildUserPublishFeedKey(userID)
	if err := writeUserPublishZSet(l.ctx, l.svcCtx, feedKey, contentId, publishedAtMillis); err != nil {
		l.Logger.Errorf("更新用户发布列表缓存失败 contentId=%d: %v", contentId, err)
	}
	if shouldSeedHotIncrement(visibility) {
		if err := writePublishHotSeed(l.ctx, l.svcCtx, contentId); err != nil {
			l.Logger.Errorf("写热榜增量失败 contentId=%d: %v", contentId, err)
		}
	}
	// 推拉结合：小账号 fan-out 到 follower inbox，大 V 跳过
	fanOutToFollowersAsync(l.svcCtx, userID, contentId, publishedAtMillis, visibility)
}
