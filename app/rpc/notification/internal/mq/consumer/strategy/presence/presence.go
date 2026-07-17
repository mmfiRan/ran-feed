// Package presence 通知域出现型策略
// 复用 count presence 翻转判定思想 但产出物是 NotifyEvent 且严格遵守 N9(只在激活态生成)+N10(actor==recipient 自我过滤)
package presence

import (
	"strings"

	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
)

// isActivation 是否为「激活态生成」触发点
// INSERT 且当前活跃 → true
// UPDATE 且旧不活跃 → 新活跃 → true
// DELETE / 无翻转 / 变为不活跃 → false(通知层只做正向触达)
func isActivation(op string, isActive func(map[string]interface{}) bool, row, oldRow map[string]interface{}) bool {
	switch strings.ToUpper(strings.TrimSpace(op)) {
	case "INSERT":
		return isActive(row)
	case "UPDATE":
		return !isActive(beforeView(row, oldRow)) && isActive(row)
	default:
		return false
	}
}

// beforeView 还原变更前整行 canal 的 Old 只带改动列 用其覆盖新行得到旧视图
func beforeView(row, oldRow map[string]interface{}) map[string]interface{} {
	if len(oldRow) == 0 {
		return row
	}
	merged := make(map[string]interface{}, len(row))
	for k, v := range row {
		merged[k] = v
	}
	for k, v := range oldRow {
		merged[k] = v
	}
	return merged
}

// statusActive status=10 视为活跃 复用互动表的 status 值域(10 正常)
func statusActive(row map[string]interface{}) bool {
	v, ok := strategy.ParseInt64(row["status"])
	if !ok {
		return false
	}
	return v == 10
}

// statusActiveNotDeleted status=10 且 is_deleted=0
func statusActiveNotDeleted(row map[string]interface{}) bool {
	if !statusActive(row) {
		return false
	}
	v, ok := strategy.ParseInt64(row["is_deleted"])
	if !ok {
		return true
	}
	return v == 0
}

// alwaysActive 无 status 列的表恒活跃(favorite 硬删语义)
func alwaysActive(map[string]interface{}) bool {
	return true
}
