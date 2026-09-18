// Package utils
package utils

import (
	userenums "ran-feed/app/rpc/user/internal/common/enums"
	"ran-feed/pkg/commonpb"
	"ran-feed/pkg/enums"
)

// GenderValue 性别转统一枚举响应
func GenderValue(gender int32) *commonpb.EnumValue {
	return enums.ToCommonPB(userenums.GenderEnum(gender))
}

// UserStatusValue 用户状态转统一枚举响应
func UserStatusValue(status int32) *commonpb.EnumValue {
	return enums.ToCommonPB(userenums.UserStatusEnum(status))
}
