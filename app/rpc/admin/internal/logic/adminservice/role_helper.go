package adminservicelogic

import (
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/entity/model"
)

// buildRoleItem 角色行转出参 created_at 毫秒
func buildRoleItem(row *model.RanFeedAdminRole) *admin.RoleItem {
	if row == nil {
		return nil
	}
	return &admin.RoleItem{
		Id:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		Remark:    row.Remark,
		CreatedAt: row.CreatedAt.UnixMilli(),
	}
}
