package commentservicelogic

import "strconv"

// parseInt64 兼容 Lua 返回的 int64 或字符串两种类型 供 obj 反查路径解析缓存字段
func parseInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case string:
		if t == "" {
			return 0
		}
		i, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}
