package consumer

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// canalMessage 一条 canal binlog 消息 一次变更可含多行
type canalMessage struct {
	ID    interface{}              `json:"id"`
	Table string                   `json:"table"`
	Type  string                   `json:"type"`
	Ts    int64                    `json:"ts"`
	Data  []map[string]interface{} `json:"data"`
	Old   []map[string]interface{} `json:"old"`
}

func (m canalMessage) table() string {
	return strings.ToLower(strings.TrimSpace(m.Table))
}

func (m canalMessage) op() string {
	return strings.ToUpper(strings.TrimSpace(m.Type))
}

// updatedAt canal 时间戳秒或毫秒自适应 缺失则取当前 作 ES external version
func (m canalMessage) updatedAt() time.Time {
	if m.Ts <= 0 {
		return time.Now()
	}
	if m.Ts > 1_000_000_000_000 {
		return time.UnixMilli(m.Ts)
	}
	return time.Unix(m.Ts, 0)
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

// rowEventID 行级幂等标识 优先用行主键 缺失则用行内容加下标 保证同一行只处理一次
func rowEventID(eventID, table, op string, row map[string]interface{}, idx int) string {
	if eventID == "" {
		eventID = "unknown"
	}
	if rowID, ok := parseInt64(row["id"]); ok && rowID > 0 {
		raw := fmt.Sprintf("%s|%s|%s|%d", eventID, table, op, rowID)
		h := sha1.Sum([]byte(raw))
		return hex.EncodeToString(h[:])
	}

	rowJSON, _ := json.Marshal(row)
	raw := fmt.Sprintf("%s|%s|%s|%d|%s", eventID, table, op, idx, string(rowJSON))
	h := sha1.Sum([]byte(raw))
	return hex.EncodeToString(h[:])
}

// parseInt64 canal 列值多为字符串 兼容数值类型
func parseInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		val, err := n.Int64()
		return val, err == nil
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		val, err := strconv.ParseInt(s, 10, 64)
		return val, err == nil
	default:
		return 0, false
	}
}
