package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingIDs(t *testing.T) {
	ids := []int64{1, 2, 3, 4}
	present := map[int64]bool{1: true, 3: true}

	// 请求了但源域投影未返回(不可索引)的 id 需删除 保序
	assert.Equal(t, []int64{2, 4}, missingIDs(ids, present))

	// 全部可索引 无删除
	assert.Empty(t, missingIDs([]int64{1, 3}, present))

	// 全部缺席 全删
	assert.Equal(t, []int64{5, 6}, missingIDs([]int64{5, 6}, map[int64]bool{}))
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want int64
		ok   bool
	}{
		{name: "canal 字符串", in: "123", want: 123, ok: true},
		{name: "float64", in: float64(45), want: 45, ok: true},
		{name: "int64", in: int64(7), want: 7, ok: true},
		{name: "空串", in: "", want: 0, ok: false},
		{name: "nil", in: nil, want: 0, ok: false},
		{name: "非数字", in: "abc", want: 0, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseInt64(tt.in)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRowEventIDStable(t *testing.T) {
	row := map[string]interface{}{"id": "100"}
	a := rowEventID("evt", "ran_feed_content", "UPDATE", row, 0)
	b := rowEventID("evt", "ran_feed_content", "UPDATE", row, 0)
	assert.Equal(t, a, b)

	other := rowEventID("evt", "ran_feed_content", "UPDATE", map[string]interface{}{"id": "200"}, 0)
	assert.NotEqual(t, a, other)
}
