// Package enums notification 服务级业务枚举 实现 pkg/enums.Enum 契约
package enums

import "fmt"

// NotifyTypeEnum 通知类型 10=赞或收藏 20=评论或回复 30=关注
type NotifyTypeEnum int32

const (
	NotifyTypeUnknown      NotifyTypeEnum = 0
	NotifyTypeLikeFavorite NotifyTypeEnum = 10
	NotifyTypeCommentReply NotifyTypeEnum = 20
	NotifyTypeFollow       NotifyTypeEnum = 30
)

var notifyTypeNames = map[NotifyTypeEnum]string{
	NotifyTypeUnknown:      "UNKNOWN",
	NotifyTypeLikeFavorite: "LIKE_FAVORITE",
	NotifyTypeCommentReply: "COMMENT_REPLY",
	NotifyTypeFollow:       "FOLLOW",
}

var notifyTypeMessages = map[NotifyTypeEnum]string{
	NotifyTypeUnknown:      "未知",
	NotifyTypeLikeFavorite: "赞或收藏",
	NotifyTypeCommentReply: "评论或回复",
	NotifyTypeFollow:       "关注",
}

func (t NotifyTypeEnum) Int32() int32 {
	return int32(t)
}

func (t NotifyTypeEnum) Valid() bool {
	_, ok := notifyTypeNames[t]
	return ok
}

func (t NotifyTypeEnum) String() string {
	if name, ok := notifyTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("NotifyTypeEnum(%d)", t)
}

func (t NotifyTypeEnum) Message() string {
	if msg, ok := notifyTypeMessages[t]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", t)
}
