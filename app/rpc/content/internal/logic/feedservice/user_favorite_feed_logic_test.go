package feedservicelogic

import (
	"testing"

	"ran-feed/app/rpc/content/content"

	"github.com/stretchr/testify/assert"
)

// TestFavoriteViewerID viewer 取 viewer_id 与 owner 无关 缺失即匿名 0
func TestFavoriteViewerID(t *testing.T) {
	viewer := int64(77)
	owner := int64(88)

	assert.Equal(t, int64(0), favoriteViewerID(nil), "nil 请求按匿名")
	assert.Equal(t, int64(0), favoriteViewerID(&content.UserFavoriteFeedReq{UserId: owner}), "未传 viewer_id 按匿名 不得回退成 owner")
	assert.Equal(t, viewer, favoriteViewerID(&content.UserFavoriteFeedReq{UserId: owner, ViewerId: &viewer}), "传了 viewer_id 用访问者")
}
