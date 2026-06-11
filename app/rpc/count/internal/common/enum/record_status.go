// Package enum count 服务级枚举 实现 pkg/enum.Enum 通用契约
package enum

import "fmt"

// RecordStatus 互动记录状态 like favorite comment follow 各表共用 10=正常 20=取消或删除
type RecordStatus int32

const (
	StatusNormal    RecordStatus = 10
	StatusCancelled RecordStatus = 20
)

var recordStatusNames = map[RecordStatus]string{
	StatusNormal:    "NORMAL",
	StatusCancelled: "CANCELLED",
}

func (s RecordStatus) Int32() int32 {
	return int32(s)
}

func (s RecordStatus) Valid() bool {
	_, ok := recordStatusNames[s]
	return ok
}

func (s RecordStatus) String() string {
	if name, ok := recordStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("RecordStatus(%d)", s)
}

// IsActive 记录是否有效 正常即有效
func (s RecordStatus) IsActive() bool {
	return s == StatusNormal
}