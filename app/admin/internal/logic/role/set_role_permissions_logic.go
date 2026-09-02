// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package role

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetRolePermissionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetRolePermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetRolePermissionsLogic {
	return &SetRolePermissionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetRolePermissionsLogic) SetRolePermissions(req *types.AdminRoleSetPermissionsReq) (resp *types.AdminRoleSetPermissionsRes, err error) {
	operatorID := utils.GetContextAdminIdWithDefault(l.ctx)
	_, err = l.svcCtx.AdminRoleRpc.SetRolePermissions(l.ctx, &admin.SetRolePermissionsReq{
		RoleId:        req.RoleId,
		PermissionIds: req.PermissionIds,
		OperatorId:    operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminRoleSetPermissionsRes{}, nil
}
