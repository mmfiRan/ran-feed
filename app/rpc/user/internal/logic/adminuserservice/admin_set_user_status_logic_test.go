package adminuserservicelogic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/user/user"
)

func TestValidateUserStatusTransition(t *testing.T) {
	const canceled = user.UserStatus(30) // 注销 不在 proto enum 内

	tests := []struct {
		name     string
		cur      user.UserStatus
		target   user.UserStatus
		wantNoop bool
		wantErr  bool
	}{
		{"封禁 正常->禁用", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_DISABLED, false, false},
		{"恢复 禁用->正常", user.UserStatus_USER_STATUS_DISABLED, user.UserStatus_USER_STATUS_ACTIVE, false, false},
		{"封禁 幂等 已禁用->禁用", user.UserStatus_USER_STATUS_DISABLED, user.UserStatus_USER_STATUS_DISABLED, true, false},
		{"恢复 幂等 已正常->正常", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_ACTIVE, true, false},
		{"封禁 注销不可封禁", canceled, user.UserStatus_USER_STATUS_DISABLED, false, true},
		{"恢复 注销不可复活", canceled, user.UserStatus_USER_STATUS_ACTIVE, false, true},
		{"恢复 正常不可恢复", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_ACTIVE, true, false},
		{"不支持的目标态 UNKNOWN", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_UNKNOWN, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noop, err := validateUserStatusTransition(tt.cur, tt.target)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantNoop, noop)
		})
	}
}
