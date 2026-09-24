package admincontentservicelogic

import (
	"context"
	"errors"
	"strings"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetContentStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	reviewRepo  repositories.ContentReviewRepository
	outboxRepo  repositories.ContentOutboxRepository
}

func NewAdminSetContentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetContentStatusLogic {
	return &AdminSetContentStatusLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		reviewRepo:  repositories.NewContentReviewRepository(ctx, svcCtx.MysqlDb),
		outboxRepo:  repositories.NewContentOutboxRepository(ctx, svcCtx.MysqlDb),
	}
}

// AdminSetContentStatus 下架/恢复
func (l *AdminSetContentStatusLogic) AdminSetContentStatus(in *content.AdminSetContentStatusReq) (*content.AdminSetContentStatusRes, error) {
	target := in.GetStatus()
	from, err := l.flipSourceStatus(target)
	if err != nil {
		return nil, err
	}

	reason := ""
	if target == content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN {
		if in.Reason == nil || strings.TrimSpace(in.GetReason()) == "" {
			return nil, errorx.NewMsg("下架原因不能为空")
		}
		reason = strings.TrimSpace(in.GetReason())
	}

	row, err := l.contentRepo.AdminGetByID(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("内容不存在")
	}

	err = query.Q.Transaction(func(tx *query.Query) error {
		affected, uerr := l.contentRepo.WithTx(tx).AdminUpdateStatus(in.ContentId, int32(from), int32(target), in.GetOperatorId())
		if uerr != nil {
			return uerr
		}
		if affected == 0 {
			return errorx.NewMsg("内容不存在或状态已变更 请刷新后重试")
		}

		if rerr := l.reviewRepo.WithTx(tx).Create(&do.ContentReviewDO{
			ID:        snowflake.GenID(),
			ContentID: in.ContentId,
			Decision:  reviewDecisionFor(target),
			Reason:    reason,
			CreatedBy: in.GetOperatorId(),
			UpdatedBy: in.GetOperatorId(),
		}); rerr != nil {
			return rerr
		}
		return l.outboxRepo.WithTx(tx).CreateEvent(buildStatusEvent(target, row, reason))
	})
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新内容状态失败"))
	}

	return &content.AdminSetContentStatusRes{
		Status: utils.ContentStatusValue(int32(target)),
	}, nil
}

func (l *AdminSetContentStatusLogic) flipSourceStatus(target content.ContentStatus) (content.ContentStatus, error) {
	switch target {
	case content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN:
		return content.ContentStatus_CONTENT_STATUS_PUBLISHED, nil
	case content.ContentStatus_CONTENT_STATUS_PUBLISHED:
		return content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN, nil
	default:
		return content.ContentStatus_CONTENT_STATUS_UNSPECIFIED, errorx.NewMsg("不支持的目标状态")
	}
}

// buildStatusEvent 下架发下架事件(带原因) 恢复发恢复事件
func buildStatusEvent(target content.ContentStatus, row *model.RanFeedContent, reason string) *contentevent.ContentEvent {
	if target == content.ContentStatus_CONTENT_STATUS_PUBLISHED {
		return contentevent.NewContentRestoredEvent(row.ID, row.UserID)
	}
	return contentevent.NewContentTakenDownEvent(row.ID, row.UserID, reason)
}

// reviewDecisionFor 目标状态对应的审核记录决策
func reviewDecisionFor(target content.ContentStatus) contentEnum.ReviewDecisionEnum {
	if target == content.ContentStatus_CONTENT_STATUS_PUBLISHED {
		return contentEnum.ReviewDecisionRestored
	}
	return contentEnum.ReviewDecisionTakenDown
}
