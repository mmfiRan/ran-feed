package do

// FollowDO 关注领域对象
type FollowDO struct {
	UserID       int64
	FollowUserID int64
	Status       int32 // 10=正常 20=取消关注
	CreatedBy    int64
	UpdatedBy    int64
}
