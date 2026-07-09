package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/consts"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	roleRepo           repositories.AdminRoleRepository
	rolePermissionRepo repositories.AdminRolePermissionRepository
	userRoleRepo       repositories.AdminUserRoleRepository
}

func NewDeleteRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRoleLogic {
	return &DeleteRoleLogic{
		ctx:                ctx,
		svcCtx:             svcCtx,
		Logger:             logx.WithContext(ctx),
		roleRepo:           repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
		rolePermissionRepo: repositories.NewAdminRolePermissionRepository(ctx, svcCtx.MysqlDb),
		userRoleRepo:       repositories.NewAdminUserRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// DeleteRole 软删角色 护栏禁删 super 事务内连带物理删权限点与管理员绑定 返回受影响管理员ID
func (l *DeleteRoleLogic) DeleteRole(in *admin.DeleteRoleReq) (*admin.DeleteRoleRes, error) {
	if in.GetId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	role, err := l.roleRepo.GetByID(in.GetId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	if role == nil {
		return nil, errorx.NewMsg("角色不存在")
	}
	if role.Code == consts.SuperRoleCode {
		return nil, errorx.NewMsg("超级管理员角色不可删除")
	}

	// 删前取受影响管理员 事务提交后随删而不可查
	adminIDs, err := l.userRoleRepo.ListAdminIDsByRoleID(in.GetId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询受影响管理员失败"))
	}

	err = query.Q.Transaction(func(tx *query.Query) error {
		if _, e := l.roleRepo.WithTx(tx).SoftDelete(in.GetId(), in.GetOperatorId()); e != nil {
			return e
		}
		if _, e := l.rolePermissionRepo.WithTx(tx).DeleteByRoleID(in.GetId()); e != nil {
			return e
		}
		if _, e := l.userRoleRepo.WithTx(tx).DeleteByRoleID(in.GetId()); e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除角色失败"))
	}

	return &admin.DeleteRoleRes{AffectedAdminIds: adminIDs}, nil
}
