package consumer

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// canalMessage 一条 canal binlog 消息 一次变更可含多行
type canalMessage struct {
	ID    interface{}              `json:"id"`
	Table string                   `json:"table"`
	Type  string                   `json:"type"`
	Ts    int64                    `json:"ts"`
	Data  []map[string]interface{} `json:"data"`
}

func (m canalMessage) table() string {
	return strings.ToLower(strings.TrimSpace(m.Table))
}

func (m canalMessage) op() string {
	return strings.ToUpper(strings.TrimSpace(m.Type))
}

// eventID 取 canal 事件 id 缺失则对原始报文取 sha1 超长截断适配去重表列宽
func (m canalMessage) eventID(raw string) string {
	id := strings.TrimSpace(fmt.Sprint(m.ID))
	if id != "" && id != "<nil>" {
		if len(id) > 64 {
			return id[:64]
		}
		return id
	}
	h := sha1.Sum([]byte(raw))
	return hex.EncodeToString(h[:])
}

// rowEventID 行级幂等标识 优先用 outbox 行主键 id 缺失回退行内容
func rowEventID(eventID, table, op string, row map[string]interface{}, idx int) string {
	if eventID == "" {
		eventID = "unknown"
	}
	if rowID := parseInt64(row["id"]); rowID > 0 {
		raw := fmt.Sprintf("%s|%s|%s|%d", eventID, table, op, rowID)
		h := sha1.Sum([]byte(raw))
		return hex.EncodeToString(h[:])
	}
	rowJSON, _ := json.Marshal(row)
	raw := fmt.Sprintf("%s|%s|%s|%d|%s", eventID, table, op, idx, string(rowJSON))
	h := sha1.Sum([]byte(raw))
	return hex.EncodeToString(h[:])
}

// stringField 取 canal 行字符串字段 flatMessage 列值均为字符串
func stringField(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// parseInt64 canal 行字段转 int64 flatMessage 为字符串其余兼容
func parseInt64(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	default:
		return 0
	}
}
