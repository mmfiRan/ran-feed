package adminservicelogic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/admin/internal/entity/model"
)

func TestBuildRoleItem(t *testing.T) {
	t.Run("正常行映射 created_at 转毫秒", func(t *testing.T) {
		row := &model.RanFeedAdminRole{
			ID:        7,
			Code:      "auditor",
			Name:      "审核员",
			Remark:    "只读审核",
			CreatedAt: time.UnixMilli(1700000000000),
		}
		item := buildRoleItem(row)
		assert.NotNil(t, item)
		assert.Equal(t, int64(7), item.Id)
		assert.Equal(t, "auditor", item.Code)
		assert.Equal(t, "审核员", item.Name)
		assert.Equal(t, "只读审核", item.Remark)
		assert.Equal(t, int64(1700000000000), item.CreatedAt)
	})

	t.Run("nil 行返回 nil", func(t *testing.T) {
		assert.Nil(t, buildRoleItem(nil))
	})
}
