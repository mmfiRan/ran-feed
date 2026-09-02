package enums

import (
	"ran-feed/pkg/commonpb"
)

// ToCommonPB 业务枚举转公共 pb 枚举响应结构
func ToCommonPB[T IntEnum](e T) *commonpb.EnumValue {
	v := Value(e)
	return &commonpb.EnumValue{
		Code:    v.Code,
		Name:    v.Name,
		Message: v.Message,
	}
}
