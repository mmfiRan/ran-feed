// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package content

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviewContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReviewContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewContentLogic {
	return &ReviewContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReviewContentLogic) ReviewContent(req *types.AdminContentReviewReq) (resp *types.AdminContentReviewRes, err error) {
	if req.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	// decision 语义映射到审核决策 拒绝时才带理由
	var decision content.ReviewDecision
	switch req.Decision {
	case consts.ContentReviewApprove:
		decision = content.ReviewDecision_REVIEW_APPROVE
	case consts.ContentReviewReject:
		decision = content.ReviewDecision_REVIEW_REJECT
	default:
		return nil, errorx.NewMsg("不支持的审核决策")
	}

	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)

	rpcRes, err := l.svcCtx.ContentAdminRpc.AdminReviewContent(l.ctx, &content.AdminReviewContentReq{
		ContentId:    req.ContentId,
		Decision:     decision,
		OperatorId:   operatorID,
		RejectReason: req.RejectReason,
	})
	if err != nil {
		return nil, err
	}

	return &types.AdminContentReviewRes{
		Status: int32(rpcRes.GetStatus()),
	}, nil
}
