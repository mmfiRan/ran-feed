package enums

import "fmt"

// IsReadEnum 通知已读状态 0=未读 1=已读
type IsReadEnum int32

const (
	IsReadUnread IsReadEnum = 0
	IsReadRead   IsReadEnum = 1
)

var isReadNames = map[IsReadEnum]string{
	IsReadUnread: "UNREAD",
	IsReadRead:   "READ",
}

var isReadMessages = map[IsReadEnum]string{
	IsReadUnread: "未读",
	IsReadRead:   "已读",
}

func (r IsReadEnum) Int32() int32 {
	return int32(r)
}

func (r IsReadEnum) Valid() bool {
	_, ok := isReadNames[r]
	return ok
}

func (r IsReadEnum) String() string {
	if name, ok := isReadNames[r]; ok {
		return name
	}
	return fmt.Sprintf("IsReadEnum(%d)", r)
}

func (r IsReadEnum) Message() string {
	if msg, ok := isReadMessages[r]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", r)
}
