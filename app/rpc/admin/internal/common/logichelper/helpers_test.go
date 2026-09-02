package logichelper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/entity/model"
)

func TestIsSelfDisable(t *testing.T) {
	tests := []struct {
		name       string
		targetID   int64
		operatorID int64
		status     admin.AdminStatus
		want       bool
	}{
		{name: "禁用自己", targetID: 1, operatorID: 1, status: admin.AdminStatus_ADMIN_DISABLED, want: true},
		{name: "禁用他人", targetID: 2, operatorID: 1, status: admin.AdminStatus_ADMIN_DISABLED, want: false},
		{name: "启用自己不算", targetID: 1, operatorID: 1, status: admin.AdminStatus_ADMIN_ENABLED, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSelfDisable(tt.targetID, tt.operatorID, tt.status))
		})
	}
}

func TestRemovesSelfSuper(t *testing.T) {
	const superID = int64(9)
	tests := []struct {
		name        string
		adminID     int64
		operatorID  int64
		superRoleID int64
		hasSuper    bool
		newRoleIDs  []int64
		want        bool
	}{
		{name: "自己移除自己super", adminID: 1, operatorID: 1, superRoleID: superID, hasSuper: true, newRoleIDs: []int64{2, 3}, want: true},
		{name: "自己保留super", adminID: 1, operatorID: 1, superRoleID: superID, hasSuper: true, newRoleIDs: []int64{superID, 2}, want: false},
		{name: "自己本无super", adminID: 1, operatorID: 1, superRoleID: superID, hasSuper: false, newRoleIDs: []int64{2}, want: false},
		{name: "改他人不触发", adminID: 2, operatorID: 1, superRoleID: superID, hasSuper: true, newRoleIDs: []int64{2}, want: false},
		{name: "无super角色不触发", adminID: 1, operatorID: 1, superRoleID: 0, hasSuper: true, newRoleIDs: []int64{2}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RemovesSelfSuper(tt.adminID, tt.operatorID, tt.superRoleID, tt.hasSuper, tt.newRoleIDs))
		})
	}
}

func TestContainsInt64(t *testing.T) {
	assert.True(t, ContainsInt64([]int64{1, 2, 3}, 2))
	assert.False(t, ContainsInt64([]int64{1, 2, 3}, 4))
	assert.False(t, ContainsInt64(nil, 1))
}

func TestBuildRoleItem(t *testing.T) {
	t.Run("正常行映射 created_at 转毫秒", func(t *testing.T) {
		row := &model.RanFeedAdminRole{
			ID:        7,
			Code:      "auditor",
			Name:      "审核员",
			Remark:    "只读审核",
			CreatedAt: time.UnixMilli(1700000000000),
		}
		item := BuildRoleItem(row)
		assert.NotNil(t, item)
		assert.Equal(t, int64(7), item.Id)
		assert.Equal(t, "auditor", item.Code)
		assert.Equal(t, "审核员", item.Name)
		assert.Equal(t, "只读审核", item.Remark)
		assert.Equal(t, int64(1700000000000), item.CreatedAt)
	})

	t.Run("nil 行返回 nil", func(t *testing.T) {
		assert.Nil(t, BuildRoleItem(nil))
	})
}
