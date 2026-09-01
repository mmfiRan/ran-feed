// Package enums count 服务级业务枚举 实现 pkg/enums.Enum 契约
package enums

import "fmt"

// BizTypeEnum 计数业务类型 10=like 20=favorite 30=comment 40=followed 41=following
type BizTypeEnum int32

const (
	BizTypeUnknown   BizTypeEnum = 0
	BizTypeLike      BizTypeEnum = 10
	BizTypeFavorite  BizTypeEnum = 20
	BizTypeComment   BizTypeEnum = 30
	BizTypeFollowed  BizTypeEnum = 40
	BizTypeFollowing BizTypeEnum = 41
)

var bizTypeNames = map[BizTypeEnum]string{
	BizTypeUnknown:   "UNKNOWN",
	BizTypeLike:      "LIKE",
	BizTypeFavorite:  "FAVORITE",
	BizTypeComment:   "COMMENT",
	BizTypeFollowed:  "FOLLOWED",
	BizTypeFollowing: "FOLLOWING",
}

var bizTypeMessages = map[BizTypeEnum]string{
	BizTypeUnknown:   "未知",
	BizTypeLike:      "点赞",
	BizTypeFavorite:  "收藏",
	BizTypeComment:   "评论",
	BizTypeFollowed:  "被关注",
	BizTypeFollowing: "关注",
}

func (b BizTypeEnum) Int32() int32 {
	return int32(b)
}

func (b BizTypeEnum) Valid() bool {
	_, ok := bizTypeNames[b]
	return ok
}

func (b BizTypeEnum) String() string {
	if name, ok := bizTypeNames[b]; ok {
		return name
	}
	return fmt.Sprintf("BizTypeEnum(%d)", b)
}

func (b BizTypeEnum) Message() string {
	if msg, ok := bizTypeMessages[b]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", b)
}
