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

type PublishVideoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepository repositories.ContentRepository
	videoRepository   repositories.VideoRepository
}

func NewPublishVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishVideoLogic {
	return &PublishVideoLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		Logger:            logx.WithContext(ctx),
		contentRepository: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		videoRepository:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *PublishVideoLogic) PublishVideo(in *content.VideoPublishReq) (*content.VideoPublishRes, error) {
	var contentId int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		videoRepo := l.videoRepository.WithTx(tx)

		contentId = snowflake.GenID()
		now := time.Now()
		contentDO := &do.ContentDO{
			ID:          contentId,
			UserID:      in.UserId,
			ContentType: int32(content.ContentType_VIDEO),
			Status:      int32(content.ContentStatus_PUBLISHED),
			Visibility:  int32(in.Visibility),
			PublishedAt: &now,
			CreatedBy:   in.UserId,
			UpdatedBy:   in.UserId,
		}
		if err := contentRepo.CreateContent(contentDO); err != nil {
			return err
		}

		videoDO := &do.VideoDO{
			ID:              snowflake.GenID(),
			ContentID:       contentId,
			MediaID:         0,
			OriginURL:       in.VideoUrl,
			CoverURL:        in.CoverUrl,
			Duration:        in.Duration,
			TranscodeStatus: 10,
		}
		return videoRepo.CreateVideo(videoDO)
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布视频失败"))
	}

	l.afterPublish(contentId, in.UserId, in.Visibility)

	return &content.VideoPublishRes{
		ContentId: contentId,
	}, nil
}

func (l *PublishVideoLogic) afterPublish(contentId, userID int64, visibility content.Visibility) {
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
}
