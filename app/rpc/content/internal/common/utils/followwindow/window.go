package followwindow

import (
	"strconv"
	"time"
)

// DefaultWindowDays 关注流 deadline 默认时间窗口天数 配置为 0 时回退
const DefaultWindowDays = 14

const millisPerDay = int64(24 * 60 * 60 * 1000)

// Days 归一窗口天数 非正回退默认
func Days(days int) int {
	if days <= 0 {
		return DefaultWindowDays
	}
	return days
}

// NowMillis 当前毫秒 用作 inbox/publish 写入 score
func NowMillis() int64 {
	return time.Now().UnixMilli()
}

// CutoffMillis 窗口下界毫秒 早于此的成员视为过期 读时不返回写时裁剪
func CutoffMillis(days int) int64 {
	return NowMillis() - int64(Days(days))*millisPerDay
}

// TTLSeconds inbox/publish 整 key 续期秒数 比窗口多留一天缓冲避免边界抖动整 key 提前消失
func TTLSeconds(days int) int {
	return (Days(days) + 1) * 24 * 60 * 60
}

// WriteArgs 组装 update/backfill 写 lua 的 ARGV keepN cutoff ttl 后跟的 score member 对
// cutoffMillis<=0 表示不按时间裁剪 inbox 传窗口下界 publish 传 0 仅靠 keepN 与 TTL 控量
// pairs 为 score1 member1 score2 member2 序列
func WriteArgs(keepN, cutoffMillis int64, ttlSeconds int, pairs ...int64) []any {
	args := make([]any, 0, 3+len(pairs))
	args = append(args,
		strconv.FormatInt(keepN, 10),
		strconv.FormatInt(cutoffMillis, 10),
		strconv.Itoa(ttlSeconds),
	)
	for _, v := range pairs {
		args = append(args, strconv.FormatInt(v, 10))
	}
	return args
}
