package consts

const (
	// TimelineKeepN 时间线统一保留条数 发布流 收藏流 关注收件箱
	TimelineKeepN int64 = 5000
	// WindowDays 时间线默认窗口天数
	WindowDays = 14
	// FollowFanOutBatchSize 发布扇出单次拉取粉丝批大小
	FollowFanOutBatchSize = 500
	// PullSetTTLSeconds 拉模式 TTL 秒
	PullSetTTLSeconds = 300
)
