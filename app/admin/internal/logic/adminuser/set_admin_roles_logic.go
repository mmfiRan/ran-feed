// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetAdminRolesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetAdminRolesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAdminRolesLogic {
	return &SetAdminRolesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetAdminRolesLogic) SetAdminRoles(req *types.AdminUserSetRolesReq) (resp *types.AdminUserSetRolesRes, err error) {
	operatorID := utils.GetContextAdminIdWithDefault(l.ctx)
	_, err = l.svcCtx.AdminUserRpc.SetAdminRoles(l.ctx, &admin.SetAdminRolesReq{
		AdminId:    req.AdminId,
		RoleIds:    req.RoleIds,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminUserSetRolesRes{}, nil
}
