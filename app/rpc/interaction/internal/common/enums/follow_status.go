package enums

import "fmt"

// FollowStatusEnum 关注状态 ran_feed_follow 表的 status 字段类型
// 业务专属 不与其他状态机共用类型 避免误用
type FollowStatusEnum int32

const (
	FollowStatusUnknown  FollowStatusEnum = 0
	FollowStatusFollow   FollowStatusEnum = 10
	FollowStatusUnfollow FollowStatusEnum = 20
)

// followStatusNames 集中维护值与名称的映射 新增值只需在此处补充
var followStatusNames = map[FollowStatusEnum]string{
	FollowStatusUnknown:  "UNKNOWN",
	FollowStatusFollow:   "FOLLOW",
	FollowStatusUnfollow: "UNFOLLOW",
}

var followStatusMessages = map[FollowStatusEnum]string{
	FollowStatusUnknown:  "未知",
	FollowStatusFollow:   "关注",
	FollowStatusUnfollow: "取消关注",
}

func (s FollowStatusEnum) Int32() int32 {
	return int32(s)
}

func (s FollowStatusEnum) Valid() bool {
	_, ok := followStatusNames[s]
	return ok
}

func (s FollowStatusEnum) String() string {
	if name, ok := followStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("FollowStatusEnum(%d)", s)
}

func (s FollowStatusEnum) Message() string {
	if msg, ok := followStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}

// IsFollowing 是否为关注态
func (s FollowStatusEnum) IsFollowing() bool {
	return s == FollowStatusFollow
}
