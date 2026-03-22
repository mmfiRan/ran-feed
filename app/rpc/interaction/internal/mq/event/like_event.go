package event

import "encoding/json"

// EventType 事件类型
type EventType string

const (
	EventTypeLike   EventType = "LIKE"
	EventTypeCancel EventType = "CANCEL"
)

// LikeEvent 点赞事件
type LikeEvent struct {
	EventID       string    `json:"event_id"`   // 事件唯一ID，用于幂等
	EventType     EventType `json:"event_type"` // 事件类型
	UserID        int64     `json:"user_id"`
	ContentID     int64     `json:"content_id"`
	ContentUserID int64     `json:"content_user_id"`
	Scene         string    `json:"scene"`
	Timestamp     int64     `json:"timestamp"` // 事件发生时间戳
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
