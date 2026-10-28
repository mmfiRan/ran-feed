// Package enums im 服务级业务枚举,实现 pkg/enums.Enum 契约
package enums

import "fmt"

// ContentTypeEnum 消息内容类型
type ContentTypeEnum int32

const (
	ContentTypeUnknown ContentTypeEnum = 0
	ContentTypeText    ContentTypeEnum = 10
	ContentTypeImage   ContentTypeEnum = 20
	ContentTypeVideo   ContentTypeEnum = 30
)

var contentTypeNames = map[ContentTypeEnum]string{
	ContentTypeUnknown: "UNKNOWN",
	ContentTypeText:    "TEXT",
	ContentTypeImage:   "IMAGE",
	ContentTypeVideo:   "VIDEO",
}

var contentTypeMessages = map[ContentTypeEnum]string{
	ContentTypeUnknown: "未知",
	ContentTypeText:    "文本",
	ContentTypeImage:   "图片",
	ContentTypeVideo:   "视频",
}

func (e ContentTypeEnum) Int32() int32 {
	return int32(e)
}

func (e ContentTypeEnum) Valid() bool {
	_, ok := contentTypeNames[e]
	return ok
}

func (e ContentTypeEnum) String() string {
	if name, ok := contentTypeNames[e]; ok {
		return name
	}
	return fmt.Sprintf("ContentTypeEnum(%d)", e)
}

func (e ContentTypeEnum) Message() string {
	if msg, ok := contentTypeMessages[e]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", e)
}
