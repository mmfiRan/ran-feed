package userservicelogic

import (
	"testing"
	"time"

	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/user"

	"github.com/stretchr/testify/assert"
)

func TestBuildUserIndexItem(t *testing.T) {
	updatedAt := time.UnixMilli(1_700_000_123_456)
	u := &model.RanFeedUser{
		ID:        7,
		Nickname:  "昵称",
		Bio:       "个人简介",
		Username:  "alice",
		Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
		UpdatedAt: updatedAt,
	}

	got := buildUserIndexItem(u)

	assert.Equal(t, int64(7), got.UserId)
	assert.Equal(t, "昵称", got.Nickname)
	assert.Equal(t, "个人简介", got.Bio)
	assert.Equal(t, "alice", got.Username)
	assert.Equal(t, user.UserStatus_USER_STATUS_ACTIVE, got.Status)
	assert.Equal(t, int64(1_700_000_123_456), got.Version)
}
