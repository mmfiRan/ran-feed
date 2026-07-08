package rbac

import "strings"

// routePermissions 路由所需权限点 key 为 "METHOD /path"
// 中间件按 r.URL.Path 精确匹配 故路由须为静态路径(不带 :id 路径参数)
// Phase B 内容管理(feat-admin-004)起激活业务门禁 后续 C/D 各模块在此登记
var routePermissions = map[string]string{
	"GET /v1/admin/contents":         "content:list",
	"GET /v1/admin/contents/detail":  "content:detail",
	"POST /v1/admin/contents/status": "content:takedown",
	"POST /v1/admin/contents/review": "content:review",
}

// RequiredPermission 返回该路由所需权限点 无登记则第二返回 false 表示只需登录
func RequiredPermission(method, path string) (string, bool) {
	code, ok := routePermissions[strings.ToUpper(method)+" "+path]
	return code, ok && code != ""
}
