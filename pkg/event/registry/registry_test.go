package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockStrategy 只实现 TableNamer 验证注册与查找
type mockStrategy struct {
	table string
}

func (m mockStrategy) TableName() string {
	return m.table
}

func TestNew_注册与查找(t *testing.T) {
	r := New(
		mockStrategy{table: "ran_feed_like"},
		mockStrategy{table: "ran_feed_follow"},
	)

	got, ok := r.Get("ran_feed_like")
	assert.True(t, ok)
	assert.Equal(t, "ran_feed_like", got.TableName())

	_, ok = r.Get("ran_feed_unknown")
	assert.False(t, ok, "未注册表应返回 false")
}

func TestGet_表名归一化(t *testing.T) {
	r := New(mockStrategy{table: " Ran_Feed_Like "})

	tests := []struct {
		name  string
		table string
	}{
		{name: "原样", table: "ran_feed_like"},
		{name: "大写", table: "RAN_FEED_LIKE"},
		{name: "带空白", table: "  ran_feed_like  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := r.Get(tt.table)
			assert.True(t, ok, "查找应对大小写与空白不敏感")
		})
	}
}

func TestNew_忽略空表名(t *testing.T) {
	r := New(mockStrategy{table: "  "}, mockStrategy{table: "ran_feed_like"})

	_, ok := r.Get("")
	assert.False(t, ok, "空表名策略不应被注册")

	_, ok = r.Get("ran_feed_like")
	assert.True(t, ok)
}

func TestNew_同表名后注册覆盖(t *testing.T) {
	first := mockStrategy{table: "ran_feed_like"}
	second := mockStrategy{table: "RAN_FEED_LIKE"}
	r := New(first, second)

	got, ok := r.Get("ran_feed_like")
	assert.True(t, ok)
	assert.Equal(t, second.table, got.table, "同表名后注册应覆盖前者")
}
