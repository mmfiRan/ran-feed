package consts

import (
	"strconv"
	"strings"
)

const (
	singleBizKeyPrefix = "S:"
	groupBizKeyPrefix  = "G:"
)

// SingleConversationBizKey 单聊幂等键,小 ID 在前保证双方算出同一个值
func SingleConversationBizKey(a, b int64) string {
	if a > b {
		a, b = b, a
	}
	return singleBizKeyPrefix + strconv.FormatInt(a, 10) + "_" + strconv.FormatInt(b, 10)
}

// GroupConversationBizKey 群聊幂等键
func GroupConversationBizKey(groupID int64) string {
	return groupBizKeyPrefix + strconv.FormatInt(groupID, 10)
}

// ParseSingleConversationBizKey 解单聊幂等键,非法返回 ok=false
func ParseSingleConversationBizKey(key string) (a, b int64, ok bool) {
	if !strings.HasPrefix(key, singleBizKeyPrefix) {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(key, singleBizKeyPrefix), "_")
	if len(parts) != 2 {
		return 0, 0, false
	}
	a, errA := strconv.ParseInt(parts[0], 10, 64)
	b, errB := strconv.ParseInt(parts[1], 10, 64)
	if errA != nil || errB != nil || a <= 0 || b <= 0 {
		return 0, 0, false
	}
	return a, b, true
}
