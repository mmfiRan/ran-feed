package adminservicelogic

import "ran-feed/app/rpc/admin/admin"

// isSelfDisable 禁用目标即操作者自己
func isSelfDisable(targetID, operatorID int64, status admin.AdminStatus) bool {
	return status == admin.AdminStatus_ADMIN_DISABLED && targetID == operatorID
}

// removesSelfSuper 操作者给自己设角色 当前持有 super 但新集合不含 super
func removesSelfSuper(adminID, operatorID, superRoleID int64, currentlyHasSuper bool, newRoleIDs []int64) bool {
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

// containsInt64 判定集合是否含目标
func containsInt64(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
