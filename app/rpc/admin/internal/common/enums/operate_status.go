package enums

import "fmt"

// OperateStatusEnum 操作结果 ran_feed_operation_log.status 1=成功 2=失败
type OperateStatusEnum int32

const (
	OperateStatusUnknown OperateStatusEnum = 0
	OperateStatusSuccess OperateStatusEnum = 1
	OperateStatusFail    OperateStatusEnum = 2
)

var operateStatusNames = map[OperateStatusEnum]string{
	OperateStatusUnknown: "UNKNOWN",
	OperateStatusSuccess: "SUCCESS",
	OperateStatusFail:    "FAIL",
}

var operateStatusMessages = map[OperateStatusEnum]string{
	OperateStatusUnknown: "未知",
	OperateStatusSuccess: "成功",
	OperateStatusFail:    "失败",
}

func (s OperateStatusEnum) Int32() int32 {
	return int32(s)
}

func (s OperateStatusEnum) Valid() bool {
	_, ok := operateStatusNames[s]
	return ok
}

func (s OperateStatusEnum) String() string {
	if name, ok := operateStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("OperateStatusEnum(%d)", s)
}

func (s OperateStatusEnum) Message() string {
	if msg, ok := operateStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}
