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

type PublishVideoLogic struct {
	ctx context.Context
	logx.Logger
	contentRepository repositories.ContentRepository
	videoRepository   repositories.VideoRepository
}

func NewPublishVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishVideoLogic {
	return &PublishVideoLogic{
		ctx:               ctx,
		Logger:            logx.WithContext(ctx),
		contentRepository: svcCtx.ContentRepository,
		videoRepository:   svcCtx.VideoRepository,
	}
}

func (l *PublishVideoLogic) PublishVideo(in *content.VideoPublishReq) (*content.VideoPublishRes, error) {
	if err := validateVideoPublish(in.Title, in.CoverUrl, in.VideoUrl); err != nil {
		return nil, err
	}
	visibility, err := resolveWriteVisibility(writeModePublish, in.Visibility, contentEnum.VisibilityUnknown)
	if err != nil {
		return nil, err
	}

	var contentId int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		videoRepo := l.videoRepository.WithTx(tx)

		contentModel := buildContentModel(in.UserId, contentEnum.ContentTypeVideo,
			contentEnum.ContentStatusPendingReview, visibility)
		contentId = contentModel.ID
		if err := contentRepo.CreateContent(l.ctx, contentModel); err != nil {
			return err
		}

		videoModel := &model.RanFeedVideo{
			ID:              snowflake.GenID(),
			ContentID:       contentId,
			Title:           in.Title,
			OriginURL:       in.VideoUrl,
			CoverURL:        in.CoverUrl,
			Duration:        in.Duration,
			TranscodeStatus: contentEnum.TranscodeStatusPending.Int32(),
		}
		return videoRepo.CreateVideo(l.ctx, videoModel)
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布视频失败"))
	}

	return &content.VideoPublishRes{
		ContentId: contentId,
	}, nil
}
