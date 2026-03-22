package event

import "encoding/json"

// EventType 事件类型
type EventType string

// EventSource 事件来源
type EventSource string

// LikeEvent 点赞事件（用于消息队列）
type LikeEvent struct {
	EventID       string      `json:"event_id"`   // 事件唯一ID，用于幂等
	EventType     EventType   `json:"event_type"` // 事件类型
	Source        EventSource `json:"source"`     // 事件来源，便于消费端区分处理
	UserID        int64       `json:"user_id"`
	ContentID     int64       `json:"content_id"`
	ContentUserID int64       `json:"content_user_id"`
	Scene         string      `json:"scene"`
	Timestamp     int64       `json:"timestamp"` // 事件发生时间戳
}

// Marshal 序列化为JSON
func (e *LikeEvent) Marshal() (string, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// UnmarshalLikeEvent 反序列化
func UnmarshalLikeEvent(data string) (*LikeEvent, error) {
	var event LikeEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, err
	}
	return &event, nil
}
