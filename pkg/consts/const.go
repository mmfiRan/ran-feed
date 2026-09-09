package consts

// CtxKey 请求上下文 key 类型，避免与其他 string 键冲突（SA1029）
type CtxKey string

const (
	// CtxKeyUserID 请求上下文中的 C 端用户ID
	CtxKeyUserID CtxKey = "user_id"
	// CtxKeyAdminID 请求上下文中的后台管理员ID
	CtxKeyAdminID CtxKey = "admin_id"
	// CtxKeyToken 请求上下文中的登录 token
	CtxKeyToken CtxKey = "token"
)
