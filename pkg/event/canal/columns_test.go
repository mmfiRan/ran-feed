package canal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOnlyIgnoredColumnsChanged 只有变更集合非空且全部落在忽略列内才返回 true
// old 缺失或不完整时保守返回 false 保证不漏处理
func TestOnlyIgnoredColumnsChanged(t *testing.T) {
	ignored := []string{"hot_score", "last_hot_score_at", "updated_at"}
	cases := []struct {
		name   string
		row    map[string]any
		oldRow map[string]any
		want   bool
	}{
		{
			name:   "仅忽略列变更",
			row:    map[string]any{"id": "1", "hot_score": "2", "updated_at": "t2"},
			oldRow: map[string]any{"id": "1", "hot_score": "1", "updated_at": "t1"},
			want:   true,
		},
		{
			name:   "含非忽略列变更",
			row:    map[string]any{"id": "1", "hot_score": "2", "status": "30"},
			oldRow: map[string]any{"id": "1", "hot_score": "1", "status": "60"},
			want:   false,
		},
		{
			name:   "无变更",
			row:    map[string]any{"id": "1", "hot_score": "1"},
			oldRow: map[string]any{"id": "1", "hot_score": "1"},
			want:   false,
		},
		{
			name:   "无 old 保守处理",
			row:    map[string]any{"id": "1", "hot_score": "2"},
			oldRow: nil,
			want:   false,
		},
		{
			name:   "old 不完整保守处理",
			row:    map[string]any{"id": "1", "hot_score": "2", "status": "30"},
			oldRow: map[string]any{"id": "1"},
			want:   false,
		},
		{
			name:   "忽略列由无变有",
			row:    map[string]any{"id": "1", "last_hot_score_at": "t1"},
			oldRow: map[string]any{"id": "1"},
			want:   false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, OnlyIgnoredColumnsChanged(tt.row, tt.oldRow, ignored...))
		})
	}
}
