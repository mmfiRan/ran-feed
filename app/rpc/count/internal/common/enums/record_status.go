// Package enums count 服务级业务枚举 实现 pkg/enums.Enum 通用契约
package enums

import "fmt"

// RecordStatusEnum 互动记录状态 like favorite comment follow 各表共用 10=正常 20=取消或删除
type RecordStatusEnum int32

const (
	RecordStatusNormal    RecordStatusEnum = 10
	RecordStatusCancelled RecordStatusEnum = 20
)

var recordStatusNames = map[RecordStatusEnum]string{
	RecordStatusNormal:    "NORMAL",
	RecordStatusCancelled: "CANCELLED",
}

var recordStatusMessages = map[RecordStatusEnum]string{
	RecordStatusNormal:    "正常",
	RecordStatusCancelled: "取消",
}

func (s RecordStatusEnum) Int32() int32 {
	return int32(s)
}

func (s RecordStatusEnum) Valid() bool {
	_, ok := recordStatusNames[s]
	return ok
}

func (s RecordStatusEnum) String() string {
	if name, ok := recordStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("RecordStatusEnum(%d)", s)
}

func (s RecordStatusEnum) Message() string {
	if msg, ok := recordStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}

// IsActive 记录是否有效 正常即有效
func (s RecordStatusEnum) IsActive() bool {
	return s == RecordStatusNormal
}
