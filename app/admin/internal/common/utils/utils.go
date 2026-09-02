package utils

import (
	"ran-feed/app/admin/internal/types"
	"ran-feed/pkg/commonpb"
)

// ToEnumValue pb 枚举值转 HTTP 响应统一枚举结构
func ToEnumValue(v *commonpb.EnumValue) types.EnumValue {
	if v == nil {
		return types.EnumValue{}
	}
	return types.EnumValue{
		Code:    v.GetCode(),
		Name:    v.GetName(),
		Message: v.GetMessage(),
	}
}
