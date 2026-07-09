package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/consts"
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

// ListRoles 角色分页 先统计总数为0直接返回
func (l *ListRolesLogic) ListRoles(in *admin.ListRolesReq) (*admin.ListRolesRes, error) {
	total, err := l.roleRepo.Count()
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("统计角色失败"))
	}
	res := &admin.ListRolesRes{Total: total}
	if total == 0 {
		return res, nil
	}

	offset, limit := utils.NormalizePage(int(in.GetPage()), int(in.GetPageSize()), consts.DefaultPageSize, consts.MaxPageSize)
	rows, err := l.roleRepo.List(offset, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
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
