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
		return articleRepo.CreateArticle(articleDO)
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布文章失败"))
	}

	l.afterPublish(contentId, in.UserId, in.Visibility)

	return &content.ArticlePublishRes{
		ContentId: contentId,
	}, nil
}

func (l *PublishArticleLogic) afterPublish(contentId, userID int64, visibility content.Visibility) {
	feedKey := rediskey.BuildUserPublishFeedKey(userID)
	contentIDStr := strconv.FormatInt(contentId, 10)
	if _, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.UpdateUserPublishZSetScript,
		[]string{feedKey},
		strconv.FormatInt(userPublishFeedKeepN, 10),
		contentIDStr, contentIDStr,
	); err != nil {
		l.Logger.Errorf("更新用户发布列表缓存失败 contentId=%d: %v", contentId, err)
	}
	if shouldSeedHotIncrement(visibility) {
		if err := writePublishHotSeed(l.ctx, l.svcCtx, contentId); err != nil {
			l.Logger.Errorf("写热榜增量失败 contentId=%d: %v", contentId, err)
		}
	}
	// 推拉结合：小账号 fan-out 到 follower inbox，大 V 跳过
	fanOutToFollowersAsync(l.svcCtx, userID, contentId, visibility)
}
