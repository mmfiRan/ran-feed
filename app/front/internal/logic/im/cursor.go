package im

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// conversationCursor 会话列表复合游标 payload
type conversationCursor struct {
	LastMsgAt int64 `json:"t"`
	ID        int64 `json:"i"`
}

// encodeConversationCursor 编码为 base64 JSON,前端当黑盒回传
func encodeConversationCursor(lastMsgAt, id int64) string {
	if lastMsgAt <= 0 || id <= 0 {
		return ""
	}
	raw, err := json.Marshal(conversationCursor{LastMsgAt: lastMsgAt, ID: id})
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(raw)
}

// decodeConversationCursor 非法游标按首页处理,避免客户端错传导致翻页死循环
func decodeConversationCursor(cursor string) (int64, int64) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, 0
	}
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, 0
	}
	var c conversationCursor
	if err := json.Unmarshal(raw, &c); err != nil || c.LastMsgAt <= 0 || c.ID <= 0 {
		return 0, 0
	}
	return c.LastMsgAt, c.ID
}
