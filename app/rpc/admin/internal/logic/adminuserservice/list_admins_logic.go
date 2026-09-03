package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/logichelper"
	adminutils "ran-feed/app/rpc/admin/internal/common/utils"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAdminsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
	userRoleRepo  repositories.AdminUserRoleRepository
	roleRepo      repositories.AdminRoleRepository
}

func NewListAdminsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminsLogic {
	return &ListAdminsLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
		userRoleRepo:  repositories.NewAdminUserRoleRepository(ctx, svcCtx.MysqlDb),
		roleRepo:      repositories.NewAdminRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListAdmins 管理员分页查询
func (l *ListAdminsLogic) ListAdmins(in *admin.ListAdminsReq) (*admin.ListAdminsRes, error) {
	status := int32(in.GetStatus())
	offset, limit := utils.NormalizePage[uint32, uint32](in.GetPage(), in.GetPageSize())
	rows, total, err := l.adminUserRepo.Page(status, offset, limit)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	res := &admin.ListAdminsRes{
		Total:    uint32(total),
		Page:     in.GetPage(),
		PageSize: in.GetPageSize(),
	}
	if total == 0 {
		return res, nil
	}

	roleCodesByAdmin, err := l.loadRoleCodes(rows)
	if err != nil {
		return nil, err
	}

	items := make([]*admin.AdminListItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, logichelper.BuildAdminListItem(row, roleCodesByAdmin[row.ID]))
	}
	res.Items = items
	return res, nil
}

// loadRoleCodes 批量取各管理员的角色码
func (l *ListAdminsLogic) loadRoleCodes(rows []*model.RanFeedAdminUser) (map[int64][]string, error) {
	adminIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			adminIDs = append(adminIDs, row.ID)
		}
	}
	roleIDsByAdmin, err := l.userRoleRepo.ListRoleIDsByAdminIDs(adminIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员角色失败"))
	}

	allRoleIDs := make([]int64, 0)
	for _, ids := range roleIDsByAdmin {
		allRoleIDs = append(allRoleIDs, ids...)
	}
	roles, err := l.roleRepo.ListByIDs(adminutils.Dedup(allRoleIDs))
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询角色失败"))
	}
	codeByRoleID := make(map[int64]string, len(roles))
	for _, role := range roles {
		if role != nil {
			codeByRoleID[role.ID] = role.Code
		}
	}

	out := make(map[int64][]string, len(roleIDsByAdmin))
	for adminID, ids := range roleIDsByAdmin {
		codes := make([]string, 0, len(ids))
		for _, id := range ids {
			if code, ok := codeByRoleID[id]; ok {
				codes = append(codes, code)
			}
		}
		out[adminID] = codes
	}
	return out, nil
}
