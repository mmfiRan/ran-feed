package rbac

import "ran-feed/app/admin/internal/common/consts"

// HasPermission 权限点集合是否包含 required 空 required 视为放行
func HasPermission(perms map[string]struct{}, required string) bool {
	if required == "" {
		return true
	}
	_, ok := perms[required]
	return ok
}

// toSet 成员切片转集合 剔除哨兵
func toSet(members []string) map[string]struct{} {
	set := make(map[string]struct{}, len(members))
	for _, m := range members {
		if m == consts.RedisAdminPermLoadedSentinel {
			continue
		}
		set[m] = struct{}{}
	}
	return set
}
