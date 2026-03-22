package do

// CommentDO 评论领域对象
//
// 说明：该 DO 仅承载 comment 业务核心字段，用于 logic <-> repository 之间传递。
// created_at/updated_at 等时间字段如需返回，可后续扩展。
type CommentDO struct {
	ID            int64
	ContentID     int64
	ContentUserID int64
	UserID        int64
	ReplyToUserID int64
	ParentID      int64
	RootID        int64
	Comment       string
	Status        int32
	IsDeleted     int32
	Version       int32
	CreatedBy     int64
	UpdatedBy     int64
}
