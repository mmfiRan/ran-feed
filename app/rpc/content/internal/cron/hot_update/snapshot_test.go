package hot_update

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewSnapshotID_GloballyUnique 连续生成的快照 id 必须不同
// 防止退回秒级时间戳 同秒两模式生成同一 key 导致快照残缺
func TestNewSnapshotID_GloballyUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := newSnapshotID()
		assert.NotEmpty(t, id)
		_, dup := seen[id]
		assert.Falsef(t, dup, "快照 id 重复: %s", id)
		seen[id] = struct{}{}
	}
}
