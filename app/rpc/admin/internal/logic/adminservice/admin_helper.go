package adminservicelogic

import (
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/entity/model"
)

// buildAdminListItem 管理员行加角色码转出参 created_at 毫秒
func buildAdminListItem(row *model.RanFeedAdminUser, roleCodes []string) *admin.AdminListItem {
	if row == nil {
		return nil
	}
	return &admin.AdminListItem{
		Id:        row.ID,
		Username:  row.Username,
		Nickname:  row.Nickname,
		Status:    admin.AdminStatus(row.Status),
		RoleCodes: roleCodes,
		CreatedAt: row.CreatedAt.UnixMilli(),
	}
}
