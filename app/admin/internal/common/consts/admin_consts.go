package consts

import (
	"strconv"

	"ran-feed/pkg/errorx"
)

const (
	// RedisAdminSessionPrefix 后台登录态 token 前缀 admin:session:{token}
	RedisAdminSessionPrefix = "admin:session"
	// RedisAdminSessionUIDPrefix 后台登录态 admin:session:uid:{adminId}
	RedisAdminSessionUIDPrefix = "admin:session:uid"
	// RedisAdminSessionExpireSecondsDefault 后台登录态默认过期 7 天
	RedisAdminSessionExpireSecondsDefault = 7 * 24 * 60 * 60

	// RedisAdminPermPrefix 管理员权限点集合缓存前缀 admin:perms:{adminId}
	RedisAdminPermPrefix = "admin:perms"
	// RedisAdminPermExpireSeconds 权限缓存过期 1 天 命中时滑动续期 角色/权限变更时主动失效
	RedisAdminPermExpireSeconds = 24 * 60 * 60
	// RedisAdminPermLoadedSentinel 零权限管理员的哨兵成员 区分缓存未命中与真无权限
	RedisAdminPermLoadedSentinel = "__loaded__"

	HeaderAuthorization = "Authorization"

	CtxKeyAdminID = "admin_id"
	CtxKeyToken   = "token"

	// ContentActionTakedown 内容下架 action 入参约定值
	ContentActionTakedown = "takedown"
	// ContentActionRestore 内容恢复 action 入参约定值
	ContentActionRestore = "restore"

	// ContentReviewApprove 审核通过 decision 入参约定值
	ContentReviewApprove = "approve"
	// ContentReviewReject 审核拒绝 decision 入参约定值
	ContentReviewReject = "reject"

	// AdminUserActionEnable 启用管理员 action 入参约定值
	AdminUserActionEnable = "enable"
	// AdminUserActionDisable 禁用管理员 action 入参约定值
	AdminUserActionDisable = "disable"
)

// ErrAdminNotLogin 后台未登录
var ErrAdminNotLogin = errorx.New("管理员未登录", 100201)

// ErrAdminForbidden 后台无权限
var ErrAdminForbidden = errorx.New("无操作权限", 100203)

func BuildAdminSessionKey(token string) string {
	return RedisAdminSessionPrefix + ":" + token
}

func BuildAdminSessionUIDKey(adminID int64) string {
	return RedisAdminSessionUIDPrefix + ":" + strconv.FormatInt(adminID, 10)
}

func BuildAdminPermKey(adminID int64) string {
	return RedisAdminPermPrefix + ":" + strconv.FormatInt(adminID, 10)
}
