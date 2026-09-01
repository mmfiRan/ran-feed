package enums

import "fmt"

// GenderEnum 用户性别 ran_feed_user.gender 0=未知 1=男 2=女
type GenderEnum int32

const (
	GenderUnknown GenderEnum = 0
	GenderMale    GenderEnum = 1
	GenderFemale  GenderEnum = 2
)

var genderNames = map[GenderEnum]string{
	GenderUnknown: "UNKNOWN",
	GenderMale:    "MALE",
	GenderFemale:  "FEMALE",
}

var genderMessages = map[GenderEnum]string{
	GenderUnknown: "未知",
	GenderMale:    "男",
	GenderFemale:  "女",
}

func (g GenderEnum) Int32() int32 {
	return int32(g)
}

func (g GenderEnum) Valid() bool {
	_, ok := genderNames[g]
	return ok
}

func (g GenderEnum) String() string {
	if name, ok := genderNames[g]; ok {
		return name
	}
	return fmt.Sprintf("GenderEnum(%d)", g)
}

func (g GenderEnum) Message() string {
	if msg, ok := genderMessages[g]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", g)
}
