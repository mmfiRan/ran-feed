package admincontentservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/convert"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminReviewContentLogic struct {
	ctx context.Context
	logx.Logger
	contentRepo repositories.ContentRepository
	reviewRepo  repositories.ContentReviewRepository
	outboxRepo  repositories.ContentOutboxRepository
}

func NewAdminReviewContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminReviewContentLogic {
	return &AdminReviewContentLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		contentRepo: svcCtx.ContentRepository,
		reviewRepo:  svcCtx.ContentReviewRepository,
		outboxRepo:  svcCtx.ContentOutboxRepository,
	}
}

// AdminReviewContent 内容审核
func (l *AdminReviewContentLogic) AdminReviewContent(in *content.AdminReviewContentReq) (*content.AdminReviewContentRes, error) {
	if in.Decision != content.ReviewDecision_REVIEW_DECISION_APPROVE && in.Decision != content.ReviewDecision_REVIEW_DECISION_REJECT {
		return nil, errorx.NewMsg("不支持的审核决策")
	}

	row, err := l.contentRepo.AdminGetByID(l.ctx, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("内容不存在")
	}
	if content.ContentStatus(row.Status) != content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW {
		return nil, errorx.NewMsg("内容不在待审状态")
	}

	approve := in.Decision == content.ReviewDecision_REVIEW_DECISION_APPROVE
	targetStatus := content.ContentStatus_CONTENT_STATUS_REJECTED
	reason := in.RejectReason
	if approve {
		targetStatus = content.ContentStatus_CONTENT_STATUS_PUBLISHED
		reason = ""
	}
	now := time.Now()

	err = query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepo.WithTx(tx)

		var (
			affected int64
			err      error
		)
		if approve {
			affected, err = contentRepo.AdminApproveContent(l.ctx, in.ContentId, in.OperatorId, now)
		} else {
			affected, err = contentRepo.AdminUpdateStatus(l.ctx, in.ContentId,
				int32(content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW), int32(targetStatus), in.OperatorId)
		}
		if err != nil {
			return err
		}
		if affected == 0 {
			return errorx.NewMsg("内容不在待审状态")
		}

		if err := l.reviewRepo.WithTx(tx).Create(l.ctx, &model.RanFeedContentReview{
			ID:        snowflake.GenID(),
			ContentID: in.ContentId,
			Decision:  int32(in.Decision),
			Reason:    reason,
			CreatedBy: in.OperatorId,
			UpdatedBy: in.OperatorId,
		}); err != nil {
			return err
		}

		// 写发件箱
		return l.outboxRepo.WithTx(tx).CreateEvent(l.ctx, l.buildReviewEvent(approve, row, reason))
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("审核失败"))
	}

	return &content.AdminReviewContentRes{
		Status: convert.ContentStatusValue(int32(targetStatus)),
	}, nil
}

func (l *AdminReviewContentLogic) buildReviewEvent(approve bool, row *model.RanFeedContent, reason string) *contentevent.ContentEvent {
	if approve {
		return contentevent.NewContentPublishedEvent(row.ID, row.UserID)
	}
	return contentevent.NewContentRejectedEvent(row.ID, row.UserID, reason)
}
