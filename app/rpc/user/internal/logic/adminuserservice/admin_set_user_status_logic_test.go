package adminuserservicelogic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/user/user"
)

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		name   string
		status user.UserStatus
		want   bool
	}{
		{"ACTIVE 有效", user.UserStatus_USER_STATUS_ACTIVE, true},
		{"DISABLED 有效", user.UserStatus_USER_STATUS_DISABLED, true},
		{"UNKNOWN 无效", user.UserStatus_USER_STATUS_UNKNOWN, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidStatus(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}