package adminuser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/admin/admin"
)

func TestMapStatusAction(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		wantStatus admin.AdminStatus
		wantOk     bool
	}{
		{name: "启用", action: "enable", wantStatus: admin.AdminStatus_ADMIN_ENABLED, wantOk: true},
		{name: "禁用", action: "disable", wantStatus: admin.AdminStatus_ADMIN_DISABLED, wantOk: true},
		{name: "非法", action: "ban", wantStatus: admin.AdminStatus_ADMIN_STATUS_UNKNOWN, wantOk: false},
		{name: "空", action: "", wantStatus: admin.AdminStatus_ADMIN_STATUS_UNKNOWN, wantOk: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, ok := mapStatusAction(tt.action)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}
