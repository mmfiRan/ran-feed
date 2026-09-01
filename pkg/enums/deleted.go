package enums

import "fmt"

// IsDeleted 软删除标记 适用于所有遵循 0/1 逻辑删除约定的表
type IsDeleted int32

const (
	NotDeleted IsDeleted = 0
	Deleted    IsDeleted = 1
)

var isDeletedNames = map[IsDeleted]string{
	NotDeleted: "NOT_DELETED",
	Deleted:    "DELETED",
}

var isDeletedMessages = map[IsDeleted]string{
	NotDeleted: "正常",
	Deleted:    "已删除",
}

func (d IsDeleted) Int32() int32 {
	return int32(d)
}

func (d IsDeleted) Valid() bool {
	_, ok := isDeletedNames[d]
	return ok
}

func (d IsDeleted) String() string {
	if name, ok := isDeletedNames[d]; ok {
		return name
	}
	return fmt.Sprintf("IsDeleted(%d)", d)
}

func (d IsDeleted) Message() string {
	if msg, ok := isDeletedMessages[d]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", d)
}

// IsDel 语义化判断 避免在调用方写 d == Deleted
func (d IsDeleted) IsDel() bool {
	return d == Deleted
}
