package rbac

import "strings"

// routePermissions 路由所需权限点 key 为 "METHOD /path"
// Phase A 无业务门禁 留空;Phase B/C 内容/用户管理路由在此登记所需 code
// 例 "GET /v1/admin/contents": "content:list"
var routePermissions = map[string]string{}

// RequiredPermission 返回该路由所需权限点 无登记则第二返回 false 表示只需登录
func RequiredPermission(method, path string) (string, bool) {
	code, ok := routePermissions[strings.ToUpper(method)+" "+path]
	return code, ok && code != ""
}
