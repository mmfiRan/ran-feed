package enums

import "fmt"

// LoginStatusEnum 登录结果 ran_feed_login_log.status 1=成功 2=失败
type LoginStatusEnum int32

const (
	LoginStatusUnknown LoginStatusEnum = 0
	LoginStatusSuccess LoginStatusEnum = 1
	LoginStatusFail    LoginStatusEnum = 2
)

var loginStatusNames = map[LoginStatusEnum]string{
	LoginStatusUnknown: "UNKNOWN",
	LoginStatusSuccess: "SUCCESS",
	LoginStatusFail:    "FAIL",
}

var loginStatusMessages = map[LoginStatusEnum]string{
	LoginStatusUnknown: "未知",
	LoginStatusSuccess: "成功",
	LoginStatusFail:    "失败",
}

func (s LoginStatusEnum) Int32() int32 {
	return int32(s)
}

func (s LoginStatusEnum) Valid() bool {
	_, ok := loginStatusNames[s]
	return ok
}

func (s LoginStatusEnum) String() string {
	if name, ok := loginStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("LoginStatusEnum(%d)", s)
}

func (s LoginStatusEnum) Message() string {
	if msg, ok := loginStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}
