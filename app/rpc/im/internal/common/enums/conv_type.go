package enums

import "fmt"

// ConvTypeEnum 会话类型
type ConvTypeEnum int32

const (
	ConvTypeUnknown ConvTypeEnum = 0
	ConvTypeSingle  ConvTypeEnum = 10
	ConvTypeGroup   ConvTypeEnum = 20
)

var convTypeNames = map[ConvTypeEnum]string{
	ConvTypeUnknown: "UNKNOWN",
	ConvTypeSingle:  "SINGLE",
	ConvTypeGroup:   "GROUP",
}

var convTypeMessages = map[ConvTypeEnum]string{
	ConvTypeUnknown: "未知",
	ConvTypeSingle:  "单聊",
	ConvTypeGroup:   "群聊",
}

func (e ConvTypeEnum) Int32() int32 {
	return int32(e)
}

func (e ConvTypeEnum) Valid() bool {
	_, ok := convTypeNames[e]
	return ok
}

func (e ConvTypeEnum) String() string {
	if name, ok := convTypeNames[e]; ok {
		return name
	}
	return fmt.Sprintf("ConvTypeEnum(%d)", e)
}

func (e ConvTypeEnum) Message() string {
	if msg, ok := convTypeMessages[e]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", e)
}
