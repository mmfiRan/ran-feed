package enums

import "fmt"

// GroupStatusEnum 群状态
type GroupStatusEnum int32

const (
	GroupStatusUnknown   GroupStatusEnum = 0
	GroupStatusNormal    GroupStatusEnum = 10
	GroupStatusDissolved GroupStatusEnum = 20
)

var groupStatusNames = map[GroupStatusEnum]string{
	GroupStatusUnknown:   "UNKNOWN",
	GroupStatusNormal:    "NORMAL",
	GroupStatusDissolved: "DISSOLVED",
}

var groupStatusMessages = map[GroupStatusEnum]string{
	GroupStatusUnknown:   "未知",
	GroupStatusNormal:    "正常",
	GroupStatusDissolved: "已解散",
}

func (e GroupStatusEnum) Int32() int32 {
	return int32(e)
}

func (e GroupStatusEnum) Valid() bool {
	_, ok := groupStatusNames[e]
	return ok
}

func (e GroupStatusEnum) String() string {
	if name, ok := groupStatusNames[e]; ok {
		return name
	}
	return fmt.Sprintf("GroupStatusEnum(%d)", e)
}

func (e GroupStatusEnum) Message() string {
	if msg, ok := groupStatusMessages[e]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", e)
}
