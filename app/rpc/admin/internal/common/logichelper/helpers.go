// Package logichelper admin-rpc 各 service logic 共享的响应映射与护栏辅助
package logichelper

import (
	"ran-feed/app/rpc/admin/admin"
	adminenums "ran-feed/app/rpc/admin/internal/common/enums"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/types"
	"ran-feed/pkg/commonpb"
	"ran-feed/pkg/enums"
	"ran-feed/pkg/snowflake"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuildAdminListItem 管理员行加角色码转出参
func BuildAdminListItem(row *model.RanFeedAdminUser, roleCodes []string) *admin.AdminListItem {
	if row == nil {
		return nil
	}
	return &admin.AdminListItem{
		Id:        row.ID,
		Username:  row.Username,
		Nickname:  row.Nickname,
		Status:    AdminStatusValue(row.Status),
		RoleCodes: roleCodes,
		CreatedAt: timestamppb.New(row.CreatedAt),
	}
}

// BuildRoleItem 角色行转出参
func BuildRoleItem(row *model.RanFeedAdminRole) *admin.RoleItem {
	if row == nil {
		return nil
	}
	return &admin.RoleItem{
		Id:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		Remark:    row.Remark,
		CreatedAt: timestamppb.New(row.CreatedAt),
	}
}

// AdminStatusValue 行状态转统一枚举响应
func AdminStatusValue(status int32) *commonpb.EnumValue {
	return enums.ToCommonPB(adminenums.AdminStatusEnum(status))
}

// BuildUserRoleRows 组装管理员角色绑定行 跳过非正 roleID 每行预生成雪花ID
func BuildUserRoleRows(adminID int64, roleIDs []int64, operatorID int64) []*model.RanFeedAdminUserRole {
	rows := make([]*model.RanFeedAdminUserRole, 0, len(roleIDs))
	for _, rid := range roleIDs {
		if rid <= 0 {
			continue
		}
		rows = append(rows, &model.RanFeedAdminUserRole{
			ID:          snowflake.GenID(),
			AdminUserID: adminID,
			RoleID:      rid,
			CreatedBy:   operatorID,
			UpdatedBy:   operatorID,
		})
	}
	return rows
}

// IsSelfDisable 禁用目标即操作者自己
func IsSelfDisable(targetID, operatorID int64, status admin.AdminStatus) bool {
	return status == admin.AdminStatus_ADMIN_STATUS_DISABLED && targetID == operatorID
}

// RemovesSelfSuper 操作者给自己设角色 当前持有 super 但新集合不含 super
func RemovesSelfSuper(adminID, operatorID, superRoleID int64, currentlyHasSuper bool, newRoleIDs []int64) bool {
	if adminID != operatorID || superRoleID <= 0 || !currentlyHasSuper {
		return false
	}
	for _, id := range newRoleIDs {
		if id == superRoleID {
			return false
		}
	}
	return true
}

// ContainsInt64 判定集合是否含目标
func ContainsInt64(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// BuildOperationLogItem 审计日志行转出参
func BuildOperationLogItem(row *types.OperationLogRow) *admin.OperationLogItem {
	if row == nil {
		return nil
	}
	return &admin.OperationLogItem{
		Id:        row.ID,
		AdminId:   row.AdminID,
		Username:  row.Username,
		Action:    row.Action,
		Title:     row.Title,
		Status:    OperateStatusValue(row.Status),
		ErrorMsg:  row.ErrorMsg,
		CostTime:  int64(row.CostTime),
		Ip:        row.IP,
		UserAgent: row.UserAgent,
		CreatedAt: timestamppb.New(row.CreatedAt),
	}
}

// OperateStatusValue 行状态转统一枚举响应
func OperateStatusValue(status int32) *commonpb.EnumValue {
	return enums.ToCommonPB(adminenums.OperateStatusEnum(status))
}

// LoginStatusValue 行状态转统一枚举响应
func LoginStatusValue(status int32) *commonpb.EnumValue {
	return enums.ToCommonPB(adminenums.LoginStatusEnum(status))
}
