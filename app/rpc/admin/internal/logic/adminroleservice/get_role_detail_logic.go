package adminroleservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/logichelper"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRoleDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	roleRepo           repositories.AdminRoleRepository
	rolePermissionRepo repositories.AdminRolePermissionRepository
}

func NewGetRoleDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleDetailLogic {
	return &GetRoleDetailLogic{
		ctx:                ctx,
		svcCtx:             svcCtx,
		Logger:             logx.WithContext(ctx),
		roleRepo:           repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
		rolePermissionRepo: repositories.NewAdminRolePermissionRepository(ctx, svcCtx.MysqlDb),
	}
}

// GetRoleDetail 角色详情含所绑权限点ID集合
func (l *GetRoleDetailLogic) GetRoleDetail(in *admin.GetRoleDetailReq) (*admin.GetRoleDetailRes, error) {
	role, err := l.roleRepo.GetByID(in.GetId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	if role == nil {
		return nil, errorx.NewMsg("角色不存在")
	}

	permIDs, err := l.rolePermissionRepo.ListPermissionIDsByRoleIDs([]int64{
		in.GetId(),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色权限失败"))
	}

	return &admin.GetRoleDetailRes{
		Role:          logichelper.BuildRoleItem(role),
		PermissionIds: permIDs,
	}, nil
}
