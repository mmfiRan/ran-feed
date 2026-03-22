package do

// FavoriteDO 收藏领域对象
type FavoriteDO struct {
	ID            int64
	UserID        int64
	ContentID     int64
	ContentUserID int64
	CreatedBy     int64
	UpdatedBy     int64
}
