package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
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
	if err := validateVideoPublish(in.Title, in.CoverUrl, in.VideoUrl); err != nil {
		return nil, err
	}
	visibility, err := resolveWriteVisibility(writeModePublish, in.Visibility, 0)
	if err != nil {
		return nil, err
	}

	var contentId int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		videoRepo := l.videoRepository.WithTx(tx)

		contentDO := buildContentDO(in.UserId, content.ContentType_CONTENT_TYPE_VIDEO,
			content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW, visibility)
		contentId = contentDO.ID
		if err := contentRepo.CreateContent(contentDO); err != nil {
			return err
		}

		videoDO := &do.VideoDO{
			ID:              snowflake.GenID(),
			ContentID:       contentId,
			Title:           in.Title,
			OriginURL:       in.VideoUrl,
			CoverURL:        in.CoverUrl,
			Duration:        in.Duration,
			TranscodeStatus: contentEnum.TranscodeStatusPending.Int32(),
		}
		return videoRepo.CreateVideo(videoDO)
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布视频失败"))
	}

	return &content.VideoPublishRes{
		ContentId: contentId,
	}, nil
}
