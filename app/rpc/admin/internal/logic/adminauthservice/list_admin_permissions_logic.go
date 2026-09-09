package adminauthservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAdminPermissionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	permissionRepo repositories.AdminPermissionRepository
}

func NewListAdminPermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminPermissionsLogic {
	return &ListAdminPermissionsLogic{
		ctx:            ctx,
		svcCtx:         svcCtx,
		Logger:         logx.WithContext(ctx),
		permissionRepo: repositories.NewAdminPermissionRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListAdminPermissions 查询权限集合
func (l *ListAdminPermissionsLogic) ListAdminPermissions(in *admin.ListAdminPermissionsReq) (*admin.ListAdminPermissionsRes, error) {
	codes, err := l.permissionRepo.ListCodesByAdminID(in.GetAdminId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询权限点失败"))
	}

	return &admin.ListAdminPermissionsRes{
		Codes: codes,
	}, nil
}
