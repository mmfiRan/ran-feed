package user

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/user/user"
)

func TestMapUserStatusAction(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		wantStatus user.UserStatus
		wantErr    bool
	}{
		{
			name:       "ban 映射为 DISABLED",
			action:     "ban",
			wantStatus: user.UserStatus_USER_STATUS_DISABLED,
			wantErr:    false,
		},
		{
			name:       "restore 映射为 ACTIVE",
			action:     "restore",
			wantStatus: user.UserStatus_USER_STATUS_ACTIVE,
			wantErr:    false,
		},
		{
			name:       "未知 action 返回错误",
			action:     "unknown",
			wantStatus: user.UserStatus_USER_STATUS_UNKNOWN,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapUserStatusAction(tt.action)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantStatus, got)
		})
	}
}