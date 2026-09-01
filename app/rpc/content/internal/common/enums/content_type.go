// Package enums content 服务级业务枚举 实现 pkg/enums.Enum 契约
package enums

import "fmt"

// ContentTypeEnum 内容类型 10=文章 20=视频
type ContentTypeEnum int32

const (
	ContentTypeUnknown ContentTypeEnum = 0
	ContentTypeArticle ContentTypeEnum = 10
	ContentTypeVideo   ContentTypeEnum = 20
)

var contentTypeNames = map[ContentTypeEnum]string{
	ContentTypeUnknown: "UNKNOWN",
	ContentTypeArticle: "ARTICLE",
	ContentTypeVideo:   "VIDEO",
}

var contentTypeMessages = map[ContentTypeEnum]string{
	ContentTypeUnknown: "未知",
	ContentTypeArticle: "文章",
	ContentTypeVideo:   "视频",
}

func (c ContentTypeEnum) Int32() int32 {
	return int32(c)
}

func (c ContentTypeEnum) Valid() bool {
	_, ok := contentTypeNames[c]
	return ok
}

func (c ContentTypeEnum) String() string {
	if name, ok := contentTypeNames[c]; ok {
		return name
	}
	return fmt.Sprintf("ContentTypeEnum(%d)", c)
}

func (c ContentTypeEnum) Message() string {
	if msg, ok := contentTypeMessages[c]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", c)
}
