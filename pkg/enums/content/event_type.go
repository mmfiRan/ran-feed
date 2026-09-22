// Package content
package content

import "fmt"

// EventTypeEnum content 域领域事件类型
type EventTypeEnum int32

const (
	EventTypeUnknown   EventTypeEnum = 0
	EventTypePublished EventTypeEnum = 10
	EventTypeDeleted   EventTypeEnum = 20
	EventTypeTakenDown EventTypeEnum = 30
	EventTypeRejected  EventTypeEnum = 40
	EventTypeRestored  EventTypeEnum = 50
)

var eventTypeNames = map[EventTypeEnum]string{
	EventTypeUnknown:   "UNKNOWN",
	EventTypePublished: "PUBLISHED",
	EventTypeDeleted:   "DELETED",
	EventTypeTakenDown: "TAKEN_DOWN",
	EventTypeRejected:  "REJECTED",
	EventTypeRestored:  "RESTORED",
}

var eventTypeMessages = map[EventTypeEnum]string{
	EventTypeUnknown:   "未知",
	EventTypePublished: "发布",
	EventTypeDeleted:   "删除",
	EventTypeTakenDown: "下架",
	EventTypeRejected:  "拒绝",
	EventTypeRestored:  "恢复",
}

func (e EventTypeEnum) Int32() int32 {
	return int32(e)
}

func (e EventTypeEnum) Valid() bool {
	_, ok := eventTypeNames[e]
	return ok
}

func (e EventTypeEnum) String() string {
	if name, ok := eventTypeNames[e]; ok {
		return name
	}
	return fmt.Sprintf("EventTypeEnum(%d)", e)
}

func (e EventTypeEnum) Message() string {
	if msg, ok := eventTypeMessages[e]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", e)
}
