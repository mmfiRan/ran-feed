package strategy

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 注意 本文件测 strategy 包内部 API 不 import presence 免 cycle
// 四表默认注册的验证放在 presence 包(其 init 已加载)

func TestRegistry_可注册可查(t *testing.T) {
	r := newRegistry(&fakeStrategy{table: "ran_feed_x"}, &fakeStrategy{table: "ran_feed_y"})
	_, ok := r.Get("ran_feed_x")
	assert.True(t, ok)
	_, ok = r.Get("ran_feed_y")
	assert.True(t, ok)
	_, ok = r.Get("ran_feed_z")
	assert.False(t, ok, "未注册返 false")
}

func TestRegistry_TableName大小写与空白归一(t *testing.T) {
	r := newRegistry(&fakeStrategy{table: "ran_feed_like"})
	_, ok := r.Get("  RAN_FEED_LIKE  ")
	assert.True(t, ok)
}

func TestRegistry_RegisterFactoryNil忽略(t *testing.T) {
	// 直接构造避免污染全局 factories
	r := newRegistry()
	r.register(nil)
	assert.Empty(t, r.strategies)
}

type fakeStrategy struct{ table string }

func (f *fakeStrategy) TableName() string { return f.table }
func (f *fakeStrategy) ExtractEvents(context.Context, string, map[string]interface{}, map[string]interface{}) []NotifyEvent {
	return nil
}

func TestRegistry_空TableName不注册(t *testing.T) {
	r := newRegistry(&fakeStrategy{table: "   "})
	assert.Empty(t, r.strategies)
}

func TestParseInt64(t *testing.T) {
	cases := []struct {
		in   interface{}
		want int64
		ok   bool
	}{
		{nil, 0, false},
		{int(10), 10, true},
		{int32(20), 20, true},
		{int64(30), 30, true},
		{uint(40), 40, true},
		{uint32(50), 50, true},
		{uint64(60), 60, true},
		{float64(70.9), 70, true},
		{"80", 80, true},
		{"  100  ", 100, true},
		{"", 0, false},
		{"abc", 0, false},
		{json.Number("90"), 90, true},
		{json.Number("bad"), 0, false},
		{true, 0, false},
	}
	for _, c := range cases {
		got, ok := ParseInt64(c.in)
		assert.Equal(t, c.ok, ok, "%v", c.in)
		if c.ok {
			assert.Equal(t, c.want, got)
		}
	}
}

func TestParseString(t *testing.T) {
	assert.Equal(t, "", ParseString(nil))
	assert.Equal(t, "hello", ParseString("hello"))
	assert.Equal(t, "hi", ParseString("  hi  "))
	assert.Equal(t, "123", ParseString(123))
	assert.Equal(t, "3.14", ParseString(3.14))
}

func TestNormalizeTableName(t *testing.T) {
	assert.Equal(t, "ran_feed_like", normalizeTableName("Ran_Feed_LIKE"))
	assert.Equal(t, "ran_feed_like", normalizeTableName(" ran_feed_like "))
	assert.Equal(t, "", normalizeTableName("   "))
}

func TestNewDefaultRegistry_跳过Nil工厂(t *testing.T) {
	require.NotPanics(t, func() { NewDefaultRegistry() })
}
