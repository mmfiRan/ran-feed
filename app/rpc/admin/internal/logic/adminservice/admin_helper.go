package adminservicelogic

import (
	"ran-feed/app/rpc/admin/admin"
	adminenums "ran-feed/app/rpc/admin/internal/common/enums"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/pkg/enums"
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
		Status:    adminStatusValue(row.Status),
		RoleCodes: roleCodes,
		CreatedAt: row.CreatedAt.UnixMilli(),
	}
}

// adminStatusValue 行状态转统一枚举响应
func adminStatusValue(status int32) *admin.EnumValue {
	v := enums.Value(adminenums.AdminStatusEnum(status))
	return &admin.EnumValue{
		Code:    v.Code,
		Name:    v.Name,
		Message: v.Message,
	}
}
