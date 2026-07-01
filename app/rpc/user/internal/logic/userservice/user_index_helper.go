package userservicelogic

import (
	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/user"
)

// maxIndexPageSize 全量重建单页上限 防止调用方传入过大 limit
const maxIndexPageSize = 500

// buildUserIndexItem 单条映射 用户文档原始投影 version 取 updated_at 毫秒
func buildUserIndexItem(u *model.RanFeedUser) *user.UserIndexItem {
	return &user.UserIndexItem{
		UserId:   u.ID,
		Nickname: u.Nickname,
		Bio:      u.Bio,
		Username: u.Username,
		Status:   user.UserStatus(u.Status),
		Version:  u.UpdatedAt.UnixMilli(),
	}
}
