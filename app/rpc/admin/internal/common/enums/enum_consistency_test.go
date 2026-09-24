package enums

import (
	"testing"

	"ran-feed/app/rpc/admin/admin"

	"github.com/stretchr/testify/assert"
)

// TestEnumValuesMatchProto 业务枚举取值必须与 proto 枚举一致
func TestEnumValuesMatchProto(t *testing.T) {
	tests := []struct {
		name  string
		biz   int32
		proto int32
	}{
		{"管理员状态 未指定", AdminStatusUnknown.Int32(), int32(admin.AdminStatus_ADMIN_STATUS_UNSPECIFIED)},
		{"管理员状态 启用", AdminStatusEnabled.Int32(), int32(admin.AdminStatus_ADMIN_STATUS_ENABLED)},
		{"管理员状态 禁用", AdminStatusDisabled.Int32(), int32(admin.AdminStatus_ADMIN_STATUS_DISABLED)},

		{"操作结果 未指定", OperateStatusUnknown.Int32(), int32(admin.OperateStatus_OPERATE_STATUS_UNSPECIFIED)},
		{"操作结果 成功", OperateStatusSuccess.Int32(), int32(admin.OperateStatus_OPERATE_STATUS_SUCCESS)},
		{"操作结果 失败", OperateStatusFail.Int32(), int32(admin.OperateStatus_OPERATE_STATUS_FAIL)},

		{"登录结果 未指定", LoginStatusUnknown.Int32(), int32(admin.LoginStatus_LOGIN_STATUS_UNSPECIFIED)},
		{"登录结果 成功", LoginStatusSuccess.Int32(), int32(admin.LoginStatus_LOGIN_STATUS_SUCCESS)},
		{"登录结果 失败", LoginStatusFail.Int32(), int32(admin.LoginStatus_LOGIN_STATUS_FAIL)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.proto, tt.biz)
		})
	}
}
