package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/utils"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetRolePermissionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	roleRepo           repositories.AdminRoleRepository
	rolePermissionRepo repositories.AdminRolePermissionRepository
	userRoleRepo       repositories.AdminUserRoleRepository
}

func NewSetRolePermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetRolePermissionsLogic {
	return &SetRolePermissionsLogic{
		ctx:                ctx,
		svcCtx:             svcCtx,
		Logger:             logx.WithContext(ctx),
		roleRepo:           repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
		rolePermissionRepo: repositories.NewAdminRolePermissionRepository(ctx, svcCtx.MysqlDb),
		userRoleRepo:       repositories.NewAdminUserRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// SetRolePermissions 覆盖式设角色权限点 事务内先删旧再插新 返回受影响管理员ID供上层失效缓存
func (l *SetRolePermissionsLogic) SetRolePermissions(in *admin.SetRolePermissionsReq) (*admin.SetRolePermissionsRes, error) {
	if in.GetRoleId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	role, err := l.roleRepo.GetByID(in.GetRoleId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	if role == nil {
		return nil, errorx.NewMsg("角色不存在")
	}

	permIDs := utils.Dedup(in.GetPermissionIds())
	err = query.Q.Transaction(func(tx *query.Query) error {
		if _, e := l.rolePermissionRepo.WithTx(tx).DeleteByRoleID(in.GetRoleId()); e != nil {
			return e
		}
		if len(permIDs) == 0 {
			return nil
		}
		rows := make([]*model.RanFeedAdminRolePermission, 0, len(permIDs))
		for _, pid := range permIDs {
			if pid <= 0 {
				continue
			}
			rows = append(rows, &model.RanFeedAdminRolePermission{
				ID:           snowflake.GenID(),
				RoleID:       in.GetRoleId(),
				PermissionID: pid,
				CreatedBy:    in.GetOperatorId(),
				UpdatedBy:    in.GetOperatorId(),
			})
		}
		return l.rolePermissionRepo.WithTx(tx).BatchCreate(rows)
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("设置角色权限失败"))
	}

	adminIDs, err := l.userRoleRepo.ListAdminIDsByRoleID(in.GetRoleId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询受影响管理员失败"))
	}
	return &admin.SetRolePermissionsRes{
		AffectedAdminIds: adminIDs,
	}, nil
}
