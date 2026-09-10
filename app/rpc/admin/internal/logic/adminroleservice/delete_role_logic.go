package adminroleservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/consts"
	"ran-feed/app/rpc/admin/internal/common/utils"
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

// DeleteRole 删角色 禁删 super 角色
func (l *DeleteRoleLogic) DeleteRole(in *admin.DeleteRoleReq) (*admin.DeleteRoleRes, error) {
	if in.GetId() <= 0 {
		return nil, errorx.NewMsg("角色ID参数错误")
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

	// 删前取受影响管理员
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

	// 角色删除 失效受影响管理员的权限缓存使新权限立即生效
	utils.InvalidateAdminPerms(l.ctx, l.svcCtx.Redis, adminIDs)
	return &admin.DeleteRoleRes{
		AffectedAdminIds: adminIDs,
	}, nil
}
