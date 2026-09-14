package admincontentservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminReviewContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	reviewRepo  repositories.ContentReviewRepository
}

func NewAdminReviewContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminReviewContentLogic {
	return &AdminReviewContentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		reviewRepo:  repositories.NewContentReviewRepository(ctx, svcCtx.MysqlDb),
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

		return l.reviewRepo.WithTx(tx).Create(&do.ContentReviewDO{
			ID:        snowflake.GenID(),
			ContentID: in.ContentId,
			Decision:  int32(in.Decision),
			Reason:    reason,
			CreatedBy: in.OperatorId,
			UpdatedBy: in.OperatorId,
		})
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("审核失败"))
	}

	if !approve {
		return &content.AdminReviewContentRes{
			Status: utils.ContentStatusValue(int32(targetStatus)),
		}, nil
	}

	// 通过内容此刻进 feed 触发发布
	l.svcCtx.FeedPublisher.Publish(l.ctx, in.ContentId, row.UserID, now.UnixMilli(), content.Visibility(row.Visibility))
	// 失效二级缓存
	if err = contentcache.Invalidate(l.ctx, l.svcCtx.Redis, in.ContentId); err != nil {
		l.Errorf("失效内容详情二级缓存失败 contentID=%d err=%v", in.ContentId, err)
	}

	return &content.AdminReviewContentRes{
		Status: utils.ContentStatusValue(int32(targetStatus)),
	}, nil
}
