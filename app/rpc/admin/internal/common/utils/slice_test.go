package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedup(t *testing.T) {
	tests := []struct {
		name string
		in   []int64
		want []int64
	}{
		{name: "空输入", in: nil, want: nil},
		{name: "无重复", in: []int64{1, 2, 3}, want: []int64{1, 2, 3}},
		{name: "有重复保序", in: []int64{3, 1, 3, 2, 1}, want: []int64{3, 1, 2}},
		{name: "全重复", in: []int64{5, 5, 5}, want: []int64{5}},
		{name: "含零与负数", in: []int64{0, -1, 0, 2, -1}, want: []int64{0, -1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want == nil {
				assert.Empty(t, Dedup(tt.in))
				return
			}
			assert.Equal(t, tt.want, Dedup(tt.in))
		})
	}
}

// TestDedupString 泛型支持 string 等其它可比较类型
func TestDedupString(t *testing.T) {
	in := []string{"b", "a", "b", "c", "a"}
	assert.Equal(t, []string{"b", "a", "c"}, Dedup(in))
}
