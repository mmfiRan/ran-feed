// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package role

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRoleLogic {
	return &CreateRoleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateRoleLogic) CreateRole(req *types.AdminRoleCreateReq) (resp *types.AdminRoleCreateRes, err error) {
	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	rpcRes, err := l.svcCtx.AdminRpc.CreateRole(l.ctx, &admin.CreateRoleReq{
		Code:       req.Code,
		Name:       req.Name,
		Remark:     req.Remark,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminRoleCreateRes{Id: rpcRes.GetId()}, nil
}
