package consts

// CtxKey 请求上下文 key 类型，
type CtxKey string

const (
	// CtxKeyUserID 请求上下文中的 C 端用户ID
	CtxKeyUserID CtxKey = "user_id"
	// CtxKeyAdminID 请求上下文中的后台管理员ID
	CtxKeyAdminID CtxKey = "admin_id"
	// CtxKeyToken 请求上下文中的登录 token
	CtxKeyToken CtxKey = "token"
)

// HotDirtyShards 热榜脏集合分片数
const HotDirtyShards = 64

var HotScoreOnlyColumns = []string{"hot_score", "last_hot_score_at", "updated_at"}

// BigVFollowerThreshold 大 V 粉丝数阈值
const BigVFollowerThreshold int64 = 50000
