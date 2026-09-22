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
