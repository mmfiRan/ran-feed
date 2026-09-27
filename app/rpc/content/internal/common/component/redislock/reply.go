package redislock

import "strconv"

// ReplyString 取 Eval 返回值里的字符串 Lua 的 string 经 RESP 回来可能是 string 也可能是 []byte
func ReplyString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case []byte:
		return string(t), true
	default:
		return "", false
	}
}

// ReplyInt64 取 Eval 返回值里的整数 Lua 的 number 和数字字符串都会出现 两种都认
func ReplyInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	case []byte:
		n, err := strconv.ParseInt(string(t), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
