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
	rpcRes, err := l.svcCtx.AdminRpc.SetRolePermissions(l.ctx, &admin.SetRolePermissionsReq{
		RoleId:        req.RoleId,
		PermissionIds: req.PermissionIds,
		OperatorId:    operatorID,
	})
	if err != nil {
		return nil, err
	}

	// 角色权限点变更 失效持有该角色的管理员权限缓存 使新权限立即生效
	invalidatePerms(l.ctx, l.svcCtx, rpcRes.GetAffectedAdminIds())
	return &types.AdminRoleSetPermissionsRes{}, nil
}
