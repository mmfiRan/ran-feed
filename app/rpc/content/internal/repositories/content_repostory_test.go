package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBuildHotScoreUpdateSQL CASE WHEN 批量更新语句与参数按 ids 顺序一一对应
func TestBuildHotScoreUpdateSQL(t *testing.T) {
	updatedAt := time.Unix(0, 0)
	sql, args := buildHotScoreUpdateSQL([]int64{1, 2}, []float64{1.5, 2.5}, updatedAt)

	assert.Equal(t,
		"UPDATE ran_feed_content SET hot_score = CASE id WHEN ? THEN ? WHEN ? THEN ? END, last_hot_score_at = ? WHERE is_deleted = 0 AND id IN (?,?)",
		sql)
	assert.Equal(t, []any{int64(1), 1.5, int64(2), 2.5, updatedAt, int64(1), int64(2)}, args)
}
