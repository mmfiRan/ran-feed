package admincontentservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminReviewContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	reviewRepo  repositories.ContentReviewRepository
	outboxRepo  repositories.ContentOutboxRepository
}

func NewAdminReviewContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminReviewContentLogic {
	return &AdminReviewContentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		reviewRepo:  repositories.NewContentReviewRepository(ctx, svcCtx.MysqlDb),
		outboxRepo:  repositories.NewContentOutboxRepository(ctx, svcCtx.MysqlDb),
	}
}

// AdminReviewContent 内容审核
func (l *AdminReviewContentLogic) AdminReviewContent(in *content.AdminReviewContentReq) (*content.AdminReviewContentRes, error) {
	if in.Decision != content.ReviewDecision_REVIEW_DECISION_APPROVE && in.Decision != content.ReviewDecision_REVIEW_DECISION_REJECT {
		return nil, errorx.NewMsg("不支持的审核决策")
	}

	row, err := l.contentRepo.AdminGetByID(in.ContentId)
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
			affected, err = contentRepo.AdminApproveContent(in.ContentId, in.OperatorId, now)
		} else {
			affected, err = contentRepo.AdminUpdateStatus(in.ContentId,
				int32(content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW), int32(targetStatus), in.OperatorId)
		}
		if err != nil {
			return err
		}
		if affected == 0 {
			return errorx.NewMsg("内容不在待审状态")
		}

		if err := l.reviewRepo.WithTx(tx).Create(&do.ContentReviewDO{
			ID:        snowflake.GenID(),
			ContentID: in.ContentId,
			Decision:  int32(in.Decision),
			Reason:    reason,
			CreatedBy: in.OperatorId,
			UpdatedBy: in.OperatorId,
		}); err != nil {
			return err
		}

		// 事务内写发件箱 通过发 ContentPublished 拒绝发 ContentRejected
		return l.outboxRepo.WithTx(tx).CreateEvent(buildReviewEvent(approve, row, now.UnixMilli(), reason))
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("审核失败"))
	}

	// 进 feed 与失效缓存的副作用由 feed 消费者消费发布事件完成
	return &content.AdminReviewContentRes{
		Status: utils.ContentStatusValue(int32(targetStatus)),
	}, nil
}

// buildReviewEvent 审核结果转 content 域事件
func buildReviewEvent(approve bool, row *model.RanFeedContent, publishedAtMillis int64, reason string) *event.ContentEvent {
	if approve {
		return &event.ContentEvent{
			EventType:   contentenums.EventTypePublished,
			ContentID:   row.ID,
			AuthorID:    row.UserID,
			ContentType: row.ContentType,
			Visibility:  row.Visibility,
			PublishedAt: publishedAtMillis,
		}
	}
	return &event.ContentEvent{
		EventType: contentenums.EventTypeRejected,
		ContentID: row.ID,
		AuthorID:  row.UserID,
		Reason:    reason,
	}
}
