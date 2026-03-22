package do

// LikeDO 点赞领域对象
type LikeDO struct {
	ID            int64
	UserID        int64
	ContentID     int64
	ContentUserID int64
	Scene         string
	Status        int32 // 10=点赞 20=取消
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
