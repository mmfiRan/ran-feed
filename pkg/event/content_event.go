// Package event
package event

import (
	"encoding/json"

	"ran-feed/pkg/enums/content"
)

// ContentEvent content 域领域事件
type ContentEvent struct {
	EventType   content.EventTypeEnum `json:"event_type"`
	ContentID   int64                 `json:"content_id"`
	AuthorID    int64                 `json:"author_id"`
	ContentType int32                 `json:"content_type"` // 10文章 20视频
	Visibility  int32                 `json:"visibility"`   // 10公开 20私密
	PublishedAt int64                 `json:"published_at"`
	Reason      string                `json:"reason"`
}

func (e *ContentEvent) Marshal() (string, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func UnmarshalContentEvent(data string) (*ContentEvent, error) {
	var e ContentEvent
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, err
	}
	return &e, nil
}
