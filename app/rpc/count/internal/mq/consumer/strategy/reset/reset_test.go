package reset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	countenum "ran-feed/app/rpc/count/internal/common/enums"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

func getResetStrategy(t *testing.T) strategy.TableStrategy {
	t.Helper()
	s, ok := strategy.NewDefaultRegistry().Get(contentTableName)
	require.True(t, ok)
	return s
}

func TestContentReset_SoftDeleteCascadesToZero(t *testing.T) {
	ctx := context.Background()
	s := getResetStrategy(t)
	const contentID, ownerID = int64(100), int64(9)

	row := map[string]interface{}{"id": contentID, "user_id": ownerID, "is_deleted": 1}
	oldRow := map[string]interface{}{"is_deleted": 0}
	updates := s.ExtractUpdates(ctx, "UPDATE", row, oldRow)

	require.Len(t, updates, 3)
	gotBiz := make(map[countenum.BizTypeEnum]bool)
	for _, u := range updates {
		assert.Equal(t, strategy.UpdateActionResetToZero, u.Action)
		assert.Equal(t, countenum.TargetTypeContent, u.TargetType)
		assert.Equal(t, contentID, u.TargetID)
		assert.Equal(t, ownerID, u.OwnerID)
		gotBiz[u.BizType] = true
	}
	assert.True(t, gotBiz[countenum.BizTypeLike])
	assert.True(t, gotBiz[countenum.BizTypeFavorite])
	assert.True(t, gotBiz[countenum.BizTypeComment])
}

func TestContentReset_IgnoresNonTransition(t *testing.T) {
	ctx := context.Background()
	s := getResetStrategy(t)

	tests := []struct {
		name   string
		op     string
		row    map[string]interface{}
		oldRow map[string]interface{}
	}{
		{name: "非更新操作忽略", op: "INSERT", row: map[string]interface{}{"id": int64(1), "is_deleted": 1}, oldRow: nil},
		{name: "is_deleted 未变忽略", op: "UPDATE", row: map[string]interface{}{"id": int64(1), "is_deleted": 1}, oldRow: map[string]interface{}{"title": "x"}},
		{name: "恢复删除 1到0 忽略", op: "UPDATE", row: map[string]interface{}{"id": int64(1), "is_deleted": 0}, oldRow: map[string]interface{}{"is_deleted": 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Empty(t, s.ExtractUpdates(ctx, tt.op, tt.row, tt.oldRow))
		})
	}
}
