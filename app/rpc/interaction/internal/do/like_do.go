package do

import "ran-feed/app/rpc/interaction/internal/common/enums"

// LikeDO 点赞领域对象
type LikeDO struct {
	ID            int64
	UserID        int64
	ContentID     int64
	ContentUserID int64
	Scene         string
	Status        enums.LikeStatus
	CreatedBy     int64
	UpdatedBy     int64
}

// LikeCountDO 点赞计数领域对象
type LikeCountDO struct {
	ID        int64
	ContentID int64
	LikeCount int64
	CreatedBy int64
	UpdatedBy int64
}
