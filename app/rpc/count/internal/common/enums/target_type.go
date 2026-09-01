package enums

import "fmt"

// TargetTypeEnum 计数对象类型 10=content 20=user
type TargetTypeEnum int32

const (
	TargetTypeUnknown TargetTypeEnum = 0
	TargetTypeContent TargetTypeEnum = 10
	TargetTypeUser    TargetTypeEnum = 20
)

var targetTypeNames = map[TargetTypeEnum]string{
	TargetTypeUnknown: "UNKNOWN",
	TargetTypeContent: "CONTENT",
	TargetTypeUser:    "USER",
}

var targetTypeMessages = map[TargetTypeEnum]string{
	TargetTypeUnknown: "未知",
	TargetTypeContent: "内容",
	TargetTypeUser:    "用户",
}

func (t TargetTypeEnum) Int32() int32 {
	return int32(t)
}

func (t TargetTypeEnum) Valid() bool {
	_, ok := targetTypeNames[t]
	return ok
}

func (t TargetTypeEnum) String() string {
	if name, ok := targetTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TargetTypeEnum(%d)", t)
}

func (t TargetTypeEnum) Message() string {
	if msg, ok := targetTypeMessages[t]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", t)
}
