package sse

import (
	"encoding/json"
	"fmt"

	"ran-feed/pkg/event/notificationevent"
)

// SSEFrame 通知流客户端帧,前端 EventSource 上按 type 分派
// type=notify 表示未读数变更,收到即走 /list 回拉
type SSEFrame struct {
	Type   string `json:"type"`
	Unread int64  `json:"unread"`
}

// EncodeMessage 把 pubsub 载荷转成 SSE 帧字节
func EncodeMessage(msg notificationevent.Payload) ([]byte, error) {
	return json.Marshal(SSEFrame{Type: "notify", Unread: msg.Unread})
}

// DecodePayload 解 pubsub 消息,空/非法 JSON/recipient<=0 返 error
func DecodePayload(raw []byte) (notificationevent.Payload, error) {
	if len(raw) == 0 {
		return notificationevent.Payload{}, fmt.Errorf("空载荷")
	}
	var msg notificationevent.Payload
	if err := json.Unmarshal(raw, &msg); err != nil {
		return notificationevent.Payload{}, fmt.Errorf("解析 pubsub payload 失败: %w", err)
	}
	if msg.RecipientID <= 0 {
		return notificationevent.Payload{}, fmt.Errorf("非法 recipient_id=%d", msg.RecipientID)
	}
	return msg, nil
}

// NotifyTransformer 通知信号载荷转 SSE 帧
// 通知只推 unread 变更信号,客户端收到即回拉 /list
func NotifyTransformer(raw []byte) ([]int64, []byte, error) {
	msg, err := DecodePayload(raw)
	if err != nil {
		return nil, nil, err
	}
	frame, err := EncodeMessage(msg)
	if err != nil {
		return nil, nil, err
	}
	return []int64{msg.RecipientID}, frame, nil
}
