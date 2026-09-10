// Package logichelper content-rpc 各 service logic 共享的统一枚举响应组装
package logichelper

import (
	contenums "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/pkg/commonpb"
	"ran-feed/pkg/enums"
)

// ContentTypeValue 内容类型转统一枚举响应
func ContentTypeValue(contentType int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contenums.ContentTypeEnum(contentType))
}

// ContentStatusValue 内容状态转统一枚举响应
func ContentStatusValue(status int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contenums.ContentStatusEnum(status))
}

// VisibilityValue 可见性转统一枚举响应
func VisibilityValue(visibility int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contenums.VisibilityEnum(visibility))
}
