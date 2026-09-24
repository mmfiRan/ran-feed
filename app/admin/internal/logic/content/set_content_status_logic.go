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

type SetContentStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetContentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetContentStatusLogic {
	return &SetContentStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetContentStatusLogic) SetContentStatus(req *types.AdminContentStatusReq) (resp *types.AdminContentStatusRes, err error) {
	operatorID := utils.GetContextAdminIdWithDefault(l.ctx)

	rpcRes, err := l.svcCtx.ContentAdminRpc.AdminSetContentStatus(l.ctx, &content.AdminSetContentStatusReq{
		ContentId:  req.ContentId,
		Status:     content.ContentStatus(req.Status),
		OperatorId: operatorID,
		Reason:     req.Reason,
	})
	if err != nil {
		return nil, err
	}

	return &types.AdminContentStatusRes{
		Status: adminutils.ToEnumValue(rpcRes.GetStatus()),
	}, nil
}
