package enums

import "fmt"

// CommentStatusEnum 评论状态 ran_feed_comment 表的 status 字段类型
// 20 是墓碑态 评论仍占位返回但不展示正文 与 is_deleted 配合使用
type CommentStatusEnum int32

const (
	CommentStatusUnknown CommentStatusEnum = 0
	CommentStatusNormal  CommentStatusEnum = 10
	CommentStatusDeleted CommentStatusEnum = 20
)

// commentStatusNames 集中维护值与名称的映射 新增值只需在此处补充
var commentStatusNames = map[CommentStatusEnum]string{
	CommentStatusUnknown: "UNKNOWN",
	CommentStatusNormal:  "NORMAL",
	CommentStatusDeleted: "DELETED",
}

var commentStatusMessages = map[CommentStatusEnum]string{
	CommentStatusUnknown: "未知",
	CommentStatusNormal:  "正常",
	CommentStatusDeleted: "已删除",
}

func (s CommentStatusEnum) Int32() int32 {
	return int32(s)
}

func (s CommentStatusEnum) Valid() bool {
	_, ok := commentStatusNames[s]
	return ok
}

func (s CommentStatusEnum) String() string {
	if name, ok := commentStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("CommentStatusEnum(%d)", s)
}

func (s CommentStatusEnum) Message() string {
	if msg, ok := commentStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}

// IsNormal 是否为正常展示态
func (s CommentStatusEnum) IsNormal() bool {
	return s == CommentStatusNormal
}

// IsDeleted 是否为墓碑态
func (s CommentStatusEnum) IsDeleted() bool {
	return s == CommentStatusDeleted
}
