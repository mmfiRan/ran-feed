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

type SetAdminRolesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
	userRoleRepo  repositories.AdminUserRoleRepository
	roleRepo      repositories.AdminRoleRepository
}

func NewSetAdminRolesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAdminRolesLogic {
	return &SetAdminRolesLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
		userRoleRepo:  repositories.NewAdminUserRoleRepository(ctx, svcCtx.MysqlDb),
		roleRepo:      repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// SetAdminRoles 覆盖式设管理员角色 护栏禁止移除自己的 super 事务内先删后建
func (l *SetAdminRolesLogic) SetAdminRoles(in *admin.SetAdminRolesReq) (*admin.SetAdminRolesRes, error) {
	if in.GetAdminId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	target, err := l.adminUserRepo.GetByID(in.GetAdminId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	if target == nil {
		return nil, errorx.NewMsg("管理员不存在")
	}

	roleIDs := dedupInt64(in.GetRoleIds())

	// 护栏 操作者给自己设角色时不得移除自己已有的 super
	if in.GetAdminId() == in.GetOperatorId() {
		superRole, err := l.roleRepo.GetByCode(consts.SuperRoleCode)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
		}
		if superRole != nil {
			currentRoleIDs, err := l.userRoleRepo.ListRoleIDsByAdminID(in.GetAdminId())
			if err != nil {
				return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员角色失败"))
			}
			hasSuper := containsInt64(currentRoleIDs, superRole.ID)
			if removesSelfSuper(in.GetAdminId(), in.GetOperatorId(), superRole.ID, hasSuper, roleIDs) {
				return nil, errorx.NewMsg("不能移除自己的超级管理员角色")
			}
		}
	}

	err = query.Q.Transaction(func(tx *query.Query) error {
		if _, e := l.userRoleRepo.WithTx(tx).DeleteByAdminID(in.GetAdminId()); e != nil {
			return e
		}
		return l.userRoleRepo.WithTx(tx).BatchCreate(buildUserRoleRows(in.GetAdminId(), roleIDs, in.GetOperatorId()))
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("设置管理员角色失败"))
	}
	return &admin.SetAdminRolesRes{}, nil
}
