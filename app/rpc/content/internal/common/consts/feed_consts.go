package consts

import "time"

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

// NowMillis 当前毫秒 时间线写入 score 统一用它
func NowMillis() int64 {
	return time.Now().UnixMilli()
}

// WindowCutoffMillis 时间线窗口下界毫秒 早于此的成员读时不返回写时裁剪
func WindowCutoffMillis() int64 {
	return NowMillis() - int64(WindowDays)*int64(time.Hour*24/time.Millisecond)
}

// TimelineTTLSeconds 时间线整 key 续期秒数 比窗口多留一天缓冲避免边界抖动整 key 提前消失
func TimelineTTLSeconds() int {
	return (WindowDays + 1) * 24 * 60 * 60
}
