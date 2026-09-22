// Package canal binlog 报文解析
package canal

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// eventIDMaxLen 去重表 event_id 超长截断
const eventIDMaxLen = 64

// tsMillisThreshold 大于该值视为毫秒时间戳
const tsMillisThreshold = 1_000_000_000_000

// Message 一条 canal binlog 消息 一次变更可能存在多行
type Message struct {
	ID       any              `json:"id"`
	RawTable string           `json:"table"`
	RawType  string           `json:"type"`
	Ts       int64            `json:"ts"`
	Data     []map[string]any `json:"data"`
	Old      []map[string]any `json:"old"`
}

// Parse 反序列化 canal 报文
func Parse(raw string) (*Message, error) {
	var msg Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *Message) Table() string {
	return strings.ToLower(strings.TrimSpace(m.RawTable))
}

func (m *Message) Op() string {
	return strings.ToUpper(strings.TrimSpace(m.RawType))
}

// OldRow 取第 idx 行的变更前状态 越界返回 nil
func (m *Message) OldRow(idx int) map[string]any {
	if idx < 0 || idx >= len(m.Old) {
		return nil
	}
	return m.Old[idx]
}

// UpdatedAt canal 时间戳秒或毫秒自适应 缺失则取当前
func (m *Message) UpdatedAt() time.Time {
	if m.Ts <= 0 {
		return time.Now()
	}
	if m.Ts > tsMillisThreshold {
		return time.UnixMilli(m.Ts)
	}
	return time.Unix(m.Ts, 0)
}

// EventID 取 canal 事件 id 缺失则对原始报文取 sha1
func (m *Message) EventID(raw string) string {
	id := strings.TrimSpace(fmt.Sprint(m.ID))
	if id != "" && id != "<nil>" {
		if len(id) > eventIDMaxLen {
			return id[:eventIDMaxLen]
		}
		return id
	}
	return sha1Hex(raw)
}

func sha1Hex(raw string) string {
	h := sha1.Sum([]byte(raw))
	return hex.EncodeToString(h[:])
}
