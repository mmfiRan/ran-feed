package presence

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsActivation(t *testing.T) {
	cases := []struct {
		name    string
		op      string
		row     map[string]interface{}
		oldRow  map[string]interface{}
		isAct   func(map[string]interface{}) bool
		want    bool
		comment string
	}{
		{"INSERT active", "INSERT", map[string]interface{}{"status": int64(10)}, nil, statusActive, true, "新增活跃产通知"},
		{"INSERT inactive", "INSERT", map[string]interface{}{"status": int64(20)}, nil, statusActive, false, "新增不活跃不产"},
		{"UPDATE inactive→active", "UPDATE", map[string]interface{}{"status": int64(10)}, map[string]interface{}{"status": int64(20)}, statusActive, true, "复活产通知"},
		{"UPDATE active→active", "UPDATE", map[string]interface{}{"status": int64(10)}, map[string]interface{}{"status": int64(10)}, statusActive, false, "已活跃不重复产"},
		{"UPDATE active→inactive", "UPDATE", map[string]interface{}{"status": int64(20)}, map[string]interface{}{"status": int64(10)}, statusActive, false, "取消不产"},
		{"DELETE active", "DELETE", map[string]interface{}{"status": int64(10)}, nil, statusActive, false, "删除不产"},
		{"unknown op", "TRUNCATE", nil, nil, statusActive, false, "未知 op 忽略"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isActivation(tc.op, tc.isAct, tc.row, tc.oldRow), tc.comment)
		})
	}
}

func TestBeforeView(t *testing.T) {
	row := map[string]interface{}{"status": int64(20), "is_deleted": int64(1), "user_id": int64(100)}
	oldRow := map[string]interface{}{"status": int64(10), "is_deleted": int64(0)}

	before := beforeView(row, oldRow)
	assert.Equal(t, int64(10), before["status"].(int64))
	assert.Equal(t, int64(0), before["is_deleted"].(int64))
	assert.Equal(t, int64(100), before["user_id"].(int64), "未变字段应保留新行值")

	// oldRow 为空时直接返回 row
	assert.Equal(t, row, beforeView(row, nil))
	assert.Equal(t, row, beforeView(row, map[string]interface{}{}))
}

func TestStatusActive(t *testing.T) {
	assert.True(t, statusActive(map[string]interface{}{"status": int64(10)}))
	assert.False(t, statusActive(map[string]interface{}{"status": int64(20)}))
	assert.False(t, statusActive(map[string]interface{}{"status": int64(30)}), "注销/软删不活跃")
	assert.False(t, statusActive(map[string]interface{}{}), "缺 status 视为不活跃")
	// canal 字符串数值兼容
	assert.True(t, statusActive(map[string]interface{}{"status": "10"}))
}

func TestStatusActiveNotDeleted(t *testing.T) {
	assert.True(t, statusActiveNotDeleted(map[string]interface{}{"status": int64(10), "is_deleted": int64(0)}))
	assert.False(t, statusActiveNotDeleted(map[string]interface{}{"status": int64(10), "is_deleted": int64(1)}))
	assert.False(t, statusActiveNotDeleted(map[string]interface{}{"status": int64(20), "is_deleted": int64(0)}))
	// is_deleted 缺失 视为未删
	assert.True(t, statusActiveNotDeleted(map[string]interface{}{"status": int64(10)}))
}

func TestAlwaysActive(t *testing.T) {
	assert.True(t, alwaysActive(nil))
	assert.True(t, alwaysActive(map[string]interface{}{}))
	assert.True(t, alwaysActive(map[string]interface{}{"any": "value"}))
}
