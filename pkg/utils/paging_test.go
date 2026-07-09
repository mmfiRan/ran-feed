package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClampPageSize(t *testing.T) {
	tests := []struct {
		name        string
		size        int
		defaultSize int
		maxSize     int
		want        int
	}{
		{name: "零取默认", size: 0, defaultSize: 20, maxSize: 100, want: 20},
		{name: "负取默认", size: -5, defaultSize: 20, maxSize: 100, want: 20},
		{name: "超上限取上限", size: 500, defaultSize: 20, maxSize: 100, want: 100},
		{name: "区间内原样", size: 50, defaultSize: 20, maxSize: 100, want: 50},
		{name: "等于上限原样", size: 100, defaultSize: 20, maxSize: 100, want: 100},
		{name: "上限0表示不设上限", size: 500, defaultSize: 20, maxSize: 0, want: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ClampPageSize(tt.size, tt.defaultSize, tt.maxSize))
		})
	}
}

func TestNormalizePage(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantOffset int
		wantLimit  int
	}{
		{name: "首页", page: 1, pageSize: 20, wantOffset: 0, wantLimit: 20},
		{name: "第三页", page: 3, pageSize: 20, wantOffset: 40, wantLimit: 20},
		{name: "页小于1归1", page: 0, pageSize: 20, wantOffset: 0, wantLimit: 20},
		{name: "负页归1", page: -2, pageSize: 20, wantOffset: 0, wantLimit: 20},
		{name: "size归一后算offset", page: 2, pageSize: 0, wantOffset: 20, wantLimit: 20},
		{name: "size超上限后算offset", page: 2, pageSize: 500, wantOffset: 100, wantLimit: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, limit := NormalizePage(tt.page, tt.pageSize, 20, 100)
			assert.Equal(t, tt.wantOffset, offset)
			assert.Equal(t, tt.wantLimit, limit)
		})
	}
}
