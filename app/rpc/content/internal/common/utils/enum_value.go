// Package utils
package utils

import (
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/pkg/commonpb"
	"ran-feed/pkg/enums"
)

// ContentTypeValue 内容类型转统一枚举响应
func ContentTypeValue(contentType int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contentEnum.ContentTypeEnum(contentType))
}

// ContentStatusValue 内容状态转统一枚举响应
func ContentStatusValue(status int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contentEnum.ContentStatusEnum(status))
}

// VisibilityValue 可见性转统一枚举响应
func VisibilityValue(visibility int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contentEnum.VisibilityEnum(visibility))
}
