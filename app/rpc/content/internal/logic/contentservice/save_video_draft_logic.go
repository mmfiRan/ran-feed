package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/consts"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveVideoDraftLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepository repositories.ContentRepository
	videoRepository   repositories.VideoRepository
}

func NewSaveVideoDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveVideoDraftLogic {
	return &SaveVideoDraftLogic{
		ctx:               ctx,
		svcCtx:            svcCtx,
		Logger:            logx.WithContext(ctx),
		contentRepository: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		videoRepository:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *SaveVideoDraftLogic) SaveVideoDraft(in *content.SaveVideoDraftReq) (*content.SaveVideoDraftRes, error) {
	if in.ContentId != nil && *in.ContentId > 0 {
		contentID, err := l.updateDraft(in, *in.ContentId)
		if err != nil {
			return nil, err
		}
		return &content.SaveVideoDraftRes{ContentId: contentID}, nil
	}

	contentID, err := l.createDraft(in)
	if err != nil {
		return nil, err
	}
	return &content.SaveVideoDraftRes{ContentId: contentID}, nil
}

func (l *SaveVideoDraftLogic) createDraft(in *content.SaveVideoDraftReq) (int64, error) {
	var contentID int64
	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		videoRepo := l.videoRepository.WithTx(tx)

		contentID = snowflake.GenID()
		contentDO := &do.ContentDO{
			ID:          contentID,
			UserID:      in.UserId,
			ContentType: int32(content.ContentType_CONTENT_TYPE_VIDEO),
			Status:      int32(content.ContentStatus_CONTENT_STATUS_DRAFT),
			Visibility:  draftVisibility(in.Visibility, int32(content.Visibility_VISIBILITY_PUBLIC)),
			CreatedBy:   in.UserId,
			UpdatedBy:   in.UserId,
		}
		if err := contentRepo.CreateContent(contentDO); err != nil {
			return err
		}
		videoDO := &do.VideoDO{
			ID:              snowflake.GenID(),
			ContentID:       contentID,
			Title:           in.Title,
			OriginURL:       in.VideoUrl,
			CoverURL:        in.CoverUrl,
			Duration:        in.Duration,
			TranscodeStatus: consts.TranscodeStatusPending,
		}
		return videoRepo.CreateVideo(videoDO)
	}); err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存草稿失败"))
	}
	return contentID, nil
}

func (l *SaveVideoDraftLogic) updateDraft(in *content.SaveVideoDraftReq, contentID int64) (int64, error) {
	row, err := l.contentRepository.GetOwnedByID(contentID, in.UserId)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存草稿失败"))
	}
	if row == nil {
		return 0, errorx.NewMsg("内容不存在或无权限")
	}
	if row.ContentType != int32(content.ContentType_CONTENT_TYPE_VIDEO) {
		return 0, errorx.NewMsg("内容类型不匹配")
	}
	if !isEditableStatus(row.Status) {
		return 0, errorx.NewMsg("该内容当前状态不可编辑")
	}

	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepository.WithTx(tx)
		videoRepo := l.videoRepository.WithTx(tx)

		if err := contentRepo.UpdateDraftMeta(contentID, draftVisibility(in.Visibility, row.Visibility), in.UserId); err != nil {
			return err
		}
		videoDO := &do.VideoDO{
			ContentID: contentID,
			Title:     in.Title,
			OriginURL: in.VideoUrl,
			CoverURL:  in.CoverUrl,
			Duration:  in.Duration,
		}
		return videoRepo.UpdateByContentID(videoDO)
	}); err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存草稿失败"))
	}
	return contentID, nil
}
