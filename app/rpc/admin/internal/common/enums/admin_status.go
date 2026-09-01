// Package enums admin
package enums

import "fmt"

// AdminStatusEnum 管理员账号状态 ran_feed_admin_user.status 10=启用 20=禁用
type AdminStatusEnum int32

const (
	AdminStatusUnknown  AdminStatusEnum = 0
	AdminStatusEnabled  AdminStatusEnum = 10
	AdminStatusDisabled AdminStatusEnum = 20
)

var adminStatusNames = map[AdminStatusEnum]string{
	AdminStatusUnknown:  "UNKNOWN",
	AdminStatusEnabled:  "ENABLED",
	AdminStatusDisabled: "DISABLED",
}

var adminStatusMessages = map[AdminStatusEnum]string{
	AdminStatusUnknown:  "未知",
	AdminStatusEnabled:  "启用",
	AdminStatusDisabled: "禁用",
}

func (s AdminStatusEnum) Int32() int32 {
	return int32(s)
}

func (s AdminStatusEnum) Valid() bool {
	_, ok := adminStatusNames[s]
	return ok
}

func (s AdminStatusEnum) String() string {
	if name, ok := adminStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("AdminStatusEnum(%d)", s)
}

func (s AdminStatusEnum) Message() string {
	if msg, ok := adminStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}
