package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     RecordStatusEnum
		wantValid  bool
		wantActive bool
	}{
		{name: "正常有效", status: RecordStatusNormal, wantValid: true, wantActive: true},
		{name: "取消无效", status: RecordStatusCancelled, wantValid: true, wantActive: false},
		{name: "未知非法", status: RecordStatusEnum(99), wantValid: false, wantActive: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantValid, tt.status.Valid())
			assert.Equal(t, tt.wantActive, tt.status.IsActive())
		})
	}
}
