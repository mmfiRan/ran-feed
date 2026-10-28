package enums

import "fmt"

// MutedEnum 会话免打扰标记
type MutedEnum int32

const (
	MutedNo  MutedEnum = 0
	MutedYes MutedEnum = 1
)

var mutedNames = map[MutedEnum]string{
	MutedNo:  "NO",
	MutedYes: "YES",
}

var mutedMessages = map[MutedEnum]string{
	MutedNo:  "否",
	MutedYes: "是",
}

func (e MutedEnum) Int32() int32 {
	return int32(e)
}

func (e MutedEnum) Valid() bool {
	_, ok := mutedNames[e]
	return ok
}

func (e MutedEnum) String() string {
	if name, ok := mutedNames[e]; ok {
		return name
	}
	return fmt.Sprintf("MutedEnum(%d)", e)
}

func (e MutedEnum) Message() string {
	if msg, ok := mutedMessages[e]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", e)
}
