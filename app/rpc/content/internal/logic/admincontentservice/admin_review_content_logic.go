package admincontentservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/logichelper"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/query"
	contentservicelogic "ran-feed/app/rpc/content/internal/logic/contentservice"
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

// AdminReviewContent 先审后发审核 仅 PENDING_REVIEW 可审
//   - 通过 转 PUBLISHED 落 published_at=审核时刻 + 事务内记审核 提交后触发进 feed 副作用
//   - 拒绝 转 REJECTED 事务内记审核(含理由) 无 feed 副作用
func (l *AdminReviewContentLogic) AdminReviewContent(in *content.AdminReviewContentReq) (*content.AdminReviewContentRes, error) {
	if in == nil || in.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
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
	now := time.Now()

	// 事务内翻状态 + 落审核记录 副作用留到提交后
	if err = query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepo.WithTx(tx)
		reviewRepo := l.reviewRepo.WithTx(tx)

		if approve {
			affected, aErr := contentRepo.AdminApproveContent(in.ContentId, in.OperatorId, now)
			if aErr != nil {
				return aErr
			}
			if affected == 0 {
				return errorx.NewMsg("内容不在待审状态")
			}
			return reviewRepo.Create(buildReviewDO(in.ContentId, int32(content.ReviewDecision_REVIEW_DECISION_APPROVE), "", in.OperatorId))
		}

		if _, uErr := contentRepo.AdminUpdateStatus(in.ContentId, int32(content.ContentStatus_CONTENT_STATUS_REJECTED), in.OperatorId); uErr != nil {
			return uErr
		}
		return reviewRepo.Create(buildReviewDO(in.ContentId, int32(content.ReviewDecision_REVIEW_DECISION_REJECT), in.RejectReason, in.OperatorId))
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("审核失败"))
	}

	if !approve {
		return &content.AdminReviewContentRes{Status: logichelper.ContentStatusValue(int32(content.ContentStatus_CONTENT_STATUS_REJECTED))}, nil
	}

	// 通过 内容此刻进 feed 触发发布副作用(publish zset + 热榜脏集合 + follower 扩散)
	contentservicelogic.RunPublishFeedEffects(l.ctx, l.svcCtx, in.ContentId, row.UserID, now.UnixMilli(), content.Visibility(row.Visibility))
	// 失效二级缓存 失败不阻断 靠 TTL 收敛
	if err = contentcache.Invalidate(l.ctx, l.svcCtx.Redis, in.ContentId); err != nil {
		l.Errorf("失效内容详情二级缓存失败 contentID=%d err=%v", in.ContentId, err)
	}

	return &content.AdminReviewContentRes{Status: logichelper.ContentStatusValue(int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED))}, nil
}

func buildReviewDO(contentID int64, decision int32, reason string, operatorID int64) *do.ContentReviewDO {
	return &do.ContentReviewDO{
		ID:        snowflake.GenID(),
		ContentID: contentID,
		Decision:  decision,
		Reason:    reason,
		CreatedBy: operatorID,
		UpdatedBy: operatorID,
	}
}
