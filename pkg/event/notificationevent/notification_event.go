// Package notificationevent 通知实时信号载荷契约,notification-rpc 发布 / front-api 订阅
package notificationevent

// Payload notify:push channel 的信号载荷,客户端收到后按未读数回拉列表
type Payload struct {
	RecipientID int64 `json:"recipient_id"`
	Unread      int64 `json:"unread"`
}
