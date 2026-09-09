package consts

import "strconv"

// SuperRoleCode 超级管理员角色码 受保护不可删除
const SuperRoleCode = "super"

// RedisAdminSessionPrefix 后台登录态 token 前缀 admin:session:{token}
const RedisAdminSessionPrefix = "admin:session"

// RedisAdminSessionUIDPrefix 后台登录态反向索引 admin:session:uid:{adminId}
const RedisAdminSessionUIDPrefix = "admin:session:uid"

// RedisAdminPermPrefix 管理员权限点集合缓存前缀 admin:perms:{adminId} 与 admin-api 保持一致
const RedisAdminPermPrefix = "admin:perms"

const DefaultSessionTTLSeconds = 7 * 24 * 60 * 60

// BuildAdminSessionKey 登录态 token 键
func BuildAdminSessionKey(token string) string {
	return RedisAdminSessionPrefix + ":" + token
}

// BuildAdminSessionUIDKey 登录态反向索引 adminId 键
func BuildAdminSessionUIDKey(adminID int64) string {
	return RedisAdminSessionUIDPrefix + ":" + strconv.FormatInt(adminID, 10)
}

// BuildAdminPermKey 管理员权限缓存键 admin:perms:{adminId}
func BuildAdminPermKey(adminID int64) string {
	return RedisAdminPermPrefix + ":" + strconv.FormatInt(adminID, 10)
}
