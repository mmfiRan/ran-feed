package presence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

func getStrategy(t *testing.T, table string) strategy.TableStrategy {
	t.Helper()
	s, ok := strategy.NewDefaultRegistry().Get(table)
	require.True(t, ok, "策略未注册 table=%s", table)
	return s
}

// findUpdate 按 bizType 取出唯一更新 便于断言
func findUpdate(t *testing.T, updates []strategy.Update, biz count.BizType) strategy.Update {
	t.Helper()
	for _, u := range updates {
		if u.BizType == biz {
			return u
		}
	}
	t.Fatalf("未找到 bizType=%d 的更新", biz)
	return strategy.Update{}
}

func TestPresenceDelta_ContentTables(t *testing.T) {
	ctx := context.Background()
	const contentID, ownerID = int64(100), int64(9)

	base := func(extra map[string]interface{}) map[string]interface{} {
		row := map[string]interface{}{"content_id": contentID, "content_user_id": ownerID}
		for k, v := range extra {
			row[k] = v
		}
		return row
	}

	tests := []struct {
		name      string
		table     string
		biz       count.BizType
		op        string
		row       map[string]interface{}
		oldRow    map[string]interface{}
		wantDelta int64 // 0 表示无更新
	}{
		{name: "点赞新增有效", table: likeTableName, biz: count.BizType_LIKE, op: "INSERT", row: base(map[string]interface{}{"status": 10}), wantDelta: 1},
		{name: "点赞新增即取消不计", table: likeTableName, biz: count.BizType_LIKE, op: "INSERT", row: base(map[string]interface{}{"status": 20}), wantDelta: 0},
		{name: "点赞删除有效减一", table: likeTableName, biz: count.BizType_LIKE, op: "DELETE", row: base(map[string]interface{}{"status": 10}), wantDelta: -1},
		{name: "点赞更新取消减一", table: likeTableName, biz: count.BizType_LIKE, op: "UPDATE", row: base(map[string]interface{}{"status": 20}), oldRow: map[string]interface{}{"status": 10}, wantDelta: -1},
		{name: "点赞更新复活加一", table: likeTableName, biz: count.BizType_LIKE, op: "UPDATE", row: base(map[string]interface{}{"status": 10}), oldRow: map[string]interface{}{"status": 20}, wantDelta: 1},
		{name: "点赞更新无翻转不计", table: likeTableName, biz: count.BizType_LIKE, op: "UPDATE", row: base(map[string]interface{}{"status": 10}), oldRow: map[string]interface{}{"content_user_id": ownerID}, wantDelta: 0},

		{name: "收藏新增加一", table: favoriteTableName, biz: count.BizType_FAVORITE, op: "INSERT", row: base(nil), wantDelta: 1},
		{name: "收藏删除减一", table: favoriteTableName, biz: count.BizType_FAVORITE, op: "DELETE", row: base(nil), wantDelta: -1},
		{name: "收藏更新不计", table: favoriteTableName, biz: count.BizType_FAVORITE, op: "UPDATE", row: base(nil), oldRow: map[string]interface{}{}, wantDelta: 0},

		{name: "评论新增有效加一", table: commentTableName, biz: count.BizType_COMMENT, op: "INSERT", row: base(map[string]interface{}{"status": 10, "is_deleted": 0}), wantDelta: 1},
		{name: "评论删除有效减一", table: commentTableName, biz: count.BizType_COMMENT, op: "DELETE", row: base(map[string]interface{}{"status": 10, "is_deleted": 0}), wantDelta: -1},
		{name: "评论删除已逻辑删不重复减", table: commentTableName, biz: count.BizType_COMMENT, op: "DELETE", row: base(map[string]interface{}{"status": 10, "is_deleted": 1}), wantDelta: 0},
		{name: "评论更新逻辑删减一", table: commentTableName, biz: count.BizType_COMMENT, op: "UPDATE", row: base(map[string]interface{}{"status": 10, "is_deleted": 1}), oldRow: map[string]interface{}{"is_deleted": 0}, wantDelta: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := getStrategy(t, tt.table)
			updates := s.ExtractUpdates(ctx, tt.op, tt.row, tt.oldRow)
			if tt.wantDelta == 0 {
				assert.Empty(t, updates)
				return
			}
			require.Len(t, updates, 1)
			u := updates[0]
			assert.Equal(t, tt.biz, u.BizType)
			assert.Equal(t, count.TargetType_CONTENT, u.TargetType)
			assert.Equal(t, contentID, u.TargetID)
			assert.Equal(t, ownerID, u.OwnerID)
			assert.Equal(t, tt.wantDelta, u.Delta)
		})
	}
}

func TestPresence_MissingContentID(t *testing.T) {
	ctx := context.Background()
	s := getStrategy(t, likeTableName)
	updates := s.ExtractUpdates(ctx, "INSERT", map[string]interface{}{"status": 10, "content_user_id": int64(9)}, nil)
	assert.Empty(t, updates)
}

func TestPresence_FollowProducesTwoTargets(t *testing.T) {
	ctx := context.Background()
	s := getStrategy(t, followTableName)
	const userID, followUserID = int64(1), int64(2)

	row := map[string]interface{}{"user_id": userID, "follow_user_id": followUserID, "status": 10, "is_deleted": 0}
	updates := s.ExtractUpdates(ctx, "INSERT", row, nil)
	require.Len(t, updates, 2)

	following := findUpdate(t, updates, count.BizType_FOLLOWING)
	assert.Equal(t, count.TargetType_USER, following.TargetType)
	assert.Equal(t, userID, following.TargetID)
	assert.Equal(t, int64(1), following.Delta)

	followed := findUpdate(t, updates, count.BizType_FOLLOWED)
	assert.Equal(t, count.TargetType_USER, followed.TargetType)
	assert.Equal(t, followUserID, followed.TargetID)
	assert.Equal(t, int64(1), followed.Delta)
}

func TestPresence_FollowCancelDecrements(t *testing.T) {
	ctx := context.Background()
	s := getStrategy(t, followTableName)
	row := map[string]interface{}{"user_id": int64(1), "follow_user_id": int64(2), "status": 20, "is_deleted": 0}
	oldRow := map[string]interface{}{"status": 10}
	updates := s.ExtractUpdates(ctx, "UPDATE", row, oldRow)
	require.Len(t, updates, 2)
	for _, u := range updates {
		assert.Equal(t, int64(-1), u.Delta)
	}
}

func TestPresence_FollowMissingUser(t *testing.T) {
	ctx := context.Background()
	s := getStrategy(t, followTableName)
	updates := s.ExtractUpdates(ctx, "INSERT", map[string]interface{}{"follow_user_id": int64(2), "status": 10}, nil)
	assert.Empty(t, updates)
}