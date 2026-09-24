package enums

import (
	"testing"

	"ran-feed/app/rpc/user/user"

	"github.com/stretchr/testify/assert"
)

// TestEnumValuesMatchProto 业务枚举取值必须与 proto 枚举一致
func TestEnumValuesMatchProto(t *testing.T) {
	tests := []struct {
		name  string
		biz   int32
		proto int32
	}{
		{"性别 未知", GenderUnknown.Int32(), int32(user.Gender_GENDER_UNSPECIFIED)},
		{"性别 男", GenderMale.Int32(), int32(user.Gender_GENDER_MALE)},
		{"性别 女", GenderFemale.Int32(), int32(user.Gender_GENDER_FEMALE)},

		{"用户状态 未知", UserStatusUnknown.Int32(), int32(user.UserStatus_USER_STATUS_UNSPECIFIED)},
		{"用户状态 正常", UserStatusActive.Int32(), int32(user.UserStatus_USER_STATUS_ACTIVE)},
		{"用户状态 禁用", UserStatusDisabled.Int32(), int32(user.UserStatus_USER_STATUS_DISABLED)},
		{"用户状态 注销", UserStatusCancelled.Int32(), int32(user.UserStatus_USER_STATUS_CANCELLED)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.proto, tt.biz)
		})
	}
}
