// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package content

import (
	"context"

	adminutils "ran-feed/app/admin/internal/common/utils"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/utils"

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
	operatorID := utils.GetContextAdminIdWithDefault(l.ctx)

	rpcRes, err := l.svcCtx.ContentAdminRpc.AdminReviewContent(l.ctx, &content.AdminReviewContentReq{
		ContentId:    req.ContentId,
		Decision:     content.ReviewDecision(req.Decision),
		OperatorId:   operatorID,
		RejectReason: req.RejectReason,
	})
	if err != nil {
		return nil, err
	}

	return &types.AdminContentReviewRes{
		Status: adminutils.ToEnumValue(rpcRes.GetStatus()),
	}, nil
}
