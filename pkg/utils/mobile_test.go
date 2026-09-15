package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeMobile(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"裸号码补默认区号", "13800138000", "+8613800138000", false},
		{"含空格横线括号", "+86 138-0013 (8000)", "+8613800138000", false},
		{"已是 E.164", "+8613800138000", "+8613800138000", false},
		{"00 国际前缀", "008613800138000", "+8613800138000", false},
		{"00 前缀境外号码", "0014155550123", "+14155550123", false},
		{"境外号码带区号", "+14155550123", "+14155550123", false},
		{"前后空白", "  13800138000  ", "+8613800138000", false},
		{"空串", "", "", true},
		{"含字母", "1380013800a", "", true},
		{"不足 E.164 最短 8 位", "12345", "", true},
		{"超过 E.164 最长 15 位", "1380013800012345", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeMobile(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
