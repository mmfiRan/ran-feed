// Package enums user 服务级业务枚举 实现 pkg/enums.Enum 契约
package enums

import "fmt"

// UserStatusEnum 用户状态 ran_feed_user.status 10=正常 20=禁用 30=注销
type UserStatusEnum int32

const (
	UserStatusUnknown   UserStatusEnum = 0
	UserStatusActive    UserStatusEnum = 10
	UserStatusDisabled  UserStatusEnum = 20
	UserStatusCancelled UserStatusEnum = 30
)

var userStatusNames = map[UserStatusEnum]string{
	UserStatusUnknown:   "UNKNOWN",
	UserStatusActive:    "ACTIVE",
	UserStatusDisabled:  "DISABLED",
	UserStatusCancelled: "CANCELLED",
}

var userStatusMessages = map[UserStatusEnum]string{
	UserStatusUnknown:   "未知",
	UserStatusActive:    "正常",
	UserStatusDisabled:  "禁用",
	UserStatusCancelled: "注销",
}

func (s UserStatusEnum) Int32() int32 {
	return int32(s)
}

func (s UserStatusEnum) Valid() bool {
	_, ok := userStatusNames[s]
	return ok
}

func (s UserStatusEnum) String() string {
	if name, ok := userStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("UserStatusEnum(%d)", s)
}

func (s UserStatusEnum) Message() string {
	if msg, ok := userStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}
