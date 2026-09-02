package adminpermissionservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPermissionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	permissionRepo repositories.AdminPermissionRepository
}

func NewListPermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPermissionsLogic {
	return &ListPermissionsLogic{
		ctx:            ctx,
		svcCtx:         svcCtx,
		Logger:         logx.WithContext(ctx),
		permissionRepo: repositories.NewAdminPermissionRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListPermissions 权限点目录 供前端角色分权时选择 module 空则全量
func (l *ListPermissionsLogic) ListPermissions(in *admin.ListPermissionsReq) (*admin.ListPermissionsRes, error) {
	rows, err := l.permissionRepo.ListAll(in.GetModule())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询权限点失败"))
	}

	items := make([]*admin.PermissionItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, &admin.PermissionItem{
			Id:     row.ID,
			Code:   row.Code,
			Name:   row.Name,
			Module: row.Module,
		})
	}
	return &admin.ListPermissionsRes{Items: items}, nil
}
