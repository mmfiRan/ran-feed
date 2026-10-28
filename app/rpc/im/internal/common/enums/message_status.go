package enums

import "fmt"

// MessageStatusEnum 消息状态
type MessageStatusEnum int32

const (
	MessageStatusUnknown MessageStatusEnum = 0
	MessageStatusNormal  MessageStatusEnum = 10
	MessageStatusRevoked MessageStatusEnum = 20
)

var messageStatusNames = map[MessageStatusEnum]string{
	MessageStatusUnknown: "UNKNOWN",
	MessageStatusNormal:  "NORMAL",
	MessageStatusRevoked: "REVOKED",
}

var messageStatusMessages = map[MessageStatusEnum]string{
	MessageStatusUnknown: "未知",
	MessageStatusNormal:  "正常",
	MessageStatusRevoked: "已撤回",
}

func (e MessageStatusEnum) Int32() int32 {
	return int32(e)
}

func (e MessageStatusEnum) Valid() bool {
	_, ok := messageStatusNames[e]
	return ok
}

func (e MessageStatusEnum) String() string {
	if name, ok := messageStatusNames[e]; ok {
		return name
	}
	return fmt.Sprintf("MessageStatusEnum(%d)", e)
}

func (e MessageStatusEnum) Message() string {
	if msg, ok := messageStatusMessages[e]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", e)
}
