// Package contentevent content 领域事件
package contentevent

import (
	"encoding/json"

	"ran-feed/pkg/enums/content"
)

// ContentEvent content 域领域事件
type ContentEvent struct {
	EventType content.EventTypeEnum `json:"event_type"`
	ContentID int64                 `json:"content_id"`
	AuthorID  int64                 `json:"author_id"`
	Reason    string                `json:"reason"`
}

// NewContentPublishedEvent 审核通过发布
func NewContentPublishedEvent(contentID, authorID int64) *ContentEvent {
	return &ContentEvent{
		EventType: content.EventTypePublished,
		ContentID: contentID,
		AuthorID:  authorID,
	}
}

// NewContentRejectedEvent 审核拒绝
func NewContentRejectedEvent(contentID, authorID int64, reason string) *ContentEvent {
	return &ContentEvent{
		EventType: content.EventTypeRejected,
		ContentID: contentID,
		AuthorID:  authorID,
		Reason:    reason,
	}
}

// NewContentDeletedEvent 作者删除
func NewContentDeletedEvent(contentID, authorID int64) *ContentEvent {
	return &ContentEvent{
		EventType: content.EventTypeDeleted,
		ContentID: contentID,
		AuthorID:  authorID,
	}
}

// NewContentTakenDownEvent 管理端下架
func NewContentTakenDownEvent(contentID, authorID int64, reason string) *ContentEvent {
	return &ContentEvent{
		EventType: content.EventTypeTakenDown,
		ContentID: contentID,
		AuthorID:  authorID,
		Reason:    reason,
	}
}

// NewContentRestoredEvent 管理端恢复上架
func NewContentRestoredEvent(contentID, authorID int64) *ContentEvent {
	return &ContentEvent{
		EventType: content.EventTypeRestored,
		ContentID: contentID,
		AuthorID:  authorID,
	}
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
