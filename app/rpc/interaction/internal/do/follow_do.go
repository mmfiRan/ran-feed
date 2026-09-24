package do

import (
	"ran-feed/app/rpc/interaction/internal/common/enums"
)

// FollowDO 关注领域对象
type FollowDO struct {
	UserID       int64
	FollowUserID int64
	Status       enums.FollowStatusEnum // 10=关注 20=取消关注
	CreatedBy    int64
	UpdatedBy    int64
}
