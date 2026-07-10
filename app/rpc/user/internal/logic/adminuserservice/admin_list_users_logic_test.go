package adminuserservicelogic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/user"
)

func TestBuildAdminUserItem(t *testing.T) {
	now := time.Now()
	row := &model.RanFeedUser{
		ID:        123,
		Username:  "testuser",
		Nickname:  "测试用户",
		Mobile:    "13800138000",
		Avatar:    "http://example.com/avatar.jpg",
		Status:    10,
		CreatedAt: now,
	}

	item := buildAdminUserItem(row)

	assert.Equal(t, int64(123), item.UserId)
	assert.Equal(t, "testuser", item.Username)
	assert.Equal(t, "测试用户", item.Nickname)
	assert.Equal(t, "13800138000", item.Mobile)
	assert.Equal(t, "http://example.com/avatar.jpg", item.Avatar)
	assert.Equal(t, user.UserStatus_USER_STATUS_ACTIVE, item.Status)
	assert.Equal(t, now.UnixMilli(), item.CreatedAt)
}