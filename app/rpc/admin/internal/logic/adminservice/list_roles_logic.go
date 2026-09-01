package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRolesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	roleRepo repositories.AdminRoleRepository
}

func NewListRolesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRolesLogic {
	return &ListRolesLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		roleRepo: repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListRoles 角色分页
func (l *ListRolesLogic) ListRoles(in *admin.ListRolesReq) (*admin.ListRolesRes, error) {
	offset, limit := utils.NormalizePage(in.GetPage(), in.GetPageSize())
	rows, total, err := l.roleRepo.Page(offset, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	res := &admin.ListRolesRes{
		Total:    uint32(total),
		Page:     in.GetPage(),
		PageSize: in.GetPageSize(),
	}
	if total == 0 {
		return res, nil
	}

	items := make([]*admin.RoleItem, 0, len(rows))
	for _, row := range rows {
		if item := buildRoleItem(row); item != nil {
			items = append(items, item)
		}
	}
	res.Items = items
	return res, nil
}
