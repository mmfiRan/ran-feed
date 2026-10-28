// Package imevent IM 实时下行帧契约,im-rpc 发布 / front-api 订阅
package imevent

import "encoding/json"

// SSE 帧类型
const (
	TypeConnected = "connected"
	TypeHeartbeat = "heartbeat"
	TypeMessage   = "message"
	TypeRevoke    = "revoke"
)

// Frame 客户端 SSE 帧,空字段不序列化
type Frame struct {
	Type           string `json:"type"`
	ConversationID int64  `json:"conversation_id,string,omitempty"`
	MessageID      int64  `json:"message_id,string,omitempty"`
	Seq            int64  `json:"seq,string,omitempty"`
	SenderID       int64  `json:"sender_id,string,omitempty"`
	ContentType    int32  `json:"content_type,omitempty"`
	Content        string `json:"content,omitempty"`
	SendTime       int64  `json:"send_time,omitempty"`
}

// Envelope im:push channel 载荷,Frame 原样透传给客户端
type Envelope struct {
	UserIDs []int64         `json:"user_ids"`
	Frame   json.RawMessage `json:"frame"`
}
