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

// CastPtr nil 安全指针类型转换 把 *U 转成 *T 未传的 nil 原样返回
func CastPtr[T ~int32, U ~int32](v *U) *T {
	if v == nil {
		return nil
	}
	t := T(*v)
	return &t
}
