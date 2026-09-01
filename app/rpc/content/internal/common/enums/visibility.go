package enums

import "fmt"

// VisibilityEnum 内容可见性 10=公开 20=私密
type VisibilityEnum int32

const (
	VisibilityUnknown VisibilityEnum = 0
	VisibilityPublic  VisibilityEnum = 10
	VisibilityPrivate VisibilityEnum = 20
)

var visibilityNames = map[VisibilityEnum]string{
	VisibilityUnknown: "UNKNOWN",
	VisibilityPublic:  "PUBLIC",
	VisibilityPrivate: "PRIVATE",
}

var visibilityMessages = map[VisibilityEnum]string{
	VisibilityUnknown: "未知",
	VisibilityPublic:  "公开",
	VisibilityPrivate: "私密",
}

func (v VisibilityEnum) Int32() int32 {
	return int32(v)
}

func (v VisibilityEnum) Valid() bool {
	_, ok := visibilityNames[v]
	return ok
}

func (v VisibilityEnum) String() string {
	if name, ok := visibilityNames[v]; ok {
		return name
	}
	return fmt.Sprintf("VisibilityEnum(%d)", v)
}

func (v VisibilityEnum) Message() string {
	if msg, ok := visibilityMessages[v]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", v)
}
