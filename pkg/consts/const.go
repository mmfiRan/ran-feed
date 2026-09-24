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

// HotDirtyShards 热榜脏集合分片数 content 与 count 两服务共用 写入与消费必须同值
const HotDirtyShards = 64

// HotScoreOnlyColumns 热榜任务分钟级落库只改这些列
// 订阅 ran_feed_content 的派生域(search/count)据此跳过与自身无关的变更
var HotScoreOnlyColumns = []string{"hot_score", "last_hot_score_at", "updated_at"}

// BigVFollowerThreshold 大 V 粉丝数阈值 由 count 侧晋升判定维护 棘轮语义只增不降
const BigVFollowerThreshold int64 = 50000
