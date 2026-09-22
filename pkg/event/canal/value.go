package canal

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseInt64 把 canal 行字段统一解析为 int64 数值
// 第二个返回值为 false 表示缺失或非法 调用方须据此跳过而非当 0 用
func ParseInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		val, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return val, true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}

// ParseString 取字符串字段 缺失或空返回空串
func ParseString(v any) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(s)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", s))
	}
}
