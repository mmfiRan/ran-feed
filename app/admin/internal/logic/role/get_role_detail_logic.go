// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package role

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRoleDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRoleDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleDetailLogic {
	return &GetRoleDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRoleDetailLogic) GetRoleDetail(req *types.AdminRoleDetailReq) (resp *types.AdminRoleDetailRes, err error) {
	rpcRes, err := l.svcCtx.AdminRpc.GetRoleDetail(l.ctx, &admin.GetRoleDetailReq{Id: req.Id})
	if err != nil {
		return nil, err
	}

	role := rpcRes.GetRole()
	return &types.AdminRoleDetailRes{
		Role: types.AdminRoleItem{
			Id:        role.GetId(),
			Code:      role.GetCode(),
			Name:      role.GetName(),
			Remark:    role.GetRemark(),
			CreatedAt: role.GetCreatedAt(),
		},
		PermissionIds: rpcRes.GetPermissionIds(),
	}, nil
}
