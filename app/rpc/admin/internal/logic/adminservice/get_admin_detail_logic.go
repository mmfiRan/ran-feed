package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAdminDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
	userRoleRepo  repositories.AdminUserRoleRepository
	roleRepo      repositories.AdminRoleRepository
}

func NewGetAdminDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminDetailLogic {
	return &GetAdminDetailLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
		userRoleRepo:  repositories.NewAdminUserRoleRepository(ctx, svcCtx.MysqlDb),
		roleRepo:      repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// GetAdminDetail 管理员详情含所绑角色ID与角色码
func (l *GetAdminDetailLogic) GetAdminDetail(in *admin.GetAdminDetailReq) (*admin.GetAdminDetailRes, error) {
	if in.GetId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	row, err := l.adminUserRepo.GetByID(in.GetId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("管理员不存在")
	}

	roleIDs, err := l.userRoleRepo.ListRoleIDsByAdminID(in.GetId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员角色失败"))
	}
	roles, err := l.roleRepo.ListByIDs(roleIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			codes = append(codes, role.Code)
		}
	}

	return &admin.GetAdminDetailRes{
		Admin:   buildAdminListItem(row, codes),
		RoleIds: roleIDs,
	}, nil
}
