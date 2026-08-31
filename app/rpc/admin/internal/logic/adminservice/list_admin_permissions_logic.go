package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/utils"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAdminPermissionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRoleRepo       repositories.AdminUserRoleRepository
	rolePermissionRepo repositories.AdminRolePermissionRepository
	permissionRepo     repositories.AdminPermissionRepository
}

func NewListAdminPermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminPermissionsLogic {
	return &ListAdminPermissionsLogic{
		ctx:                ctx,
		svcCtx:             svcCtx,
		Logger:             logx.WithContext(ctx),
		userRoleRepo:       repositories.NewAdminUserRoleRepository(ctx, svcCtx.MysqlDb),
		rolePermissionRepo: repositories.NewAdminRolePermissionRepository(ctx, svcCtx.MysqlDb),
		permissionRepo:     repositories.NewAdminPermissionRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListAdminPermissions 管理员 -> 角色 -> 权限点 code 集合 去重
func (l *ListAdminPermissionsLogic) ListAdminPermissions(in *admin.ListAdminPermissionsReq) (*admin.ListAdminPermissionsRes, error) {
	if in == nil || in.GetAdminId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	roleIDs, err := l.userRoleRepo.ListRoleIDsByAdminID(in.GetAdminId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	if len(roleIDs) == 0 {
		return &admin.ListAdminPermissionsRes{}, nil
	}

	permissionIDs, err := l.rolePermissionRepo.ListPermissionIDsByRoleIDs(roleIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询权限失败"))
	}
	if len(permissionIDs) == 0 {
		return &admin.ListAdminPermissionsRes{}, nil
	}

	codes, err := l.permissionRepo.ListCodesByIDs(utils.Dedup(permissionIDs))
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询权限点失败"))
	}

	return &admin.ListAdminPermissionsRes{
		Codes: codes,
	}, nil
}
