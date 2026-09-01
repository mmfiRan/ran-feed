package adminuser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
)

func TestToEnumValue(t *testing.T) {
	tests := []struct {
		name string
		in   *admin.EnumValue
		want types.EnumValue
	}{
		{
			name: "正常",
			in:   &admin.EnumValue{Code: 10, Name: "ENABLED", Message: "启用"},
			want: types.EnumValue{Code: 10, Name: "ENABLED", Message: "启用"},
		},
		{
			name: "nil 返回零值",
			in:   nil,
			want: types.EnumValue{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, toEnumValue(tt.in))
		})
	}
}
