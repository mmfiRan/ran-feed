//go:build integration

// 用真库跑 SQL 语义 项目条件下无 sqlmock/sqlite 且 ON DUPLICATE KEY 是 mysql-specific
// 运行: ENV_FILE=/home/wmr/opt/ran-feed-docker/.env go test -tags=integration -count=1 ./app/rpc/notification/internal/repositories/...
// 前置: ran_feed_notification 与 ran_feed_mq_consume_dedup 表已 apply

package repositories

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/pkg/envx"
)

func setupIntegration(t *testing.T) context.Context {
	t.Helper()
	envx.Load()
	if os.Getenv("MYSQL_HOST") == "" {
		t.Skip("MYSQL_HOST 未设置 跳过集成测 请 export ENV_FILE=/path/to/ran-feed-docker/.env")
	}
	g, err := gorm.Open(mysql.Open(envx.MySQLDSNFromEnv()), &gorm.Config{})
	require.NoError(t, err)
	query.SetDefault(g)

	require.NoError(t, g.Exec("TRUNCATE TABLE ran_feed_notification").Error)
	return context.Background()
}

func newRealRepo(ctx context.Context) NotificationRepository {
	return &notificationRepositoryImpl{ctx: ctx}
}

func mkNotif(recipient, actor int64, notifyType int32, aggKey string, contentID int64) *model.RanFeedNotification {
	now := time.Now()
	return &model.RanFeedNotification{
		RecipientID: recipient,
		ActorID:     actor,
		NotifyType:  notifyType,
		AggKey:      aggKey,
		AggCount:    1,
		ContentID:   contentID,
		IsRead:      0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestIntegration_UpsertAggregate_累加与ReSurface(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	// 首次 upsert
	first := mkNotif(1001, 2001, 10, "LF:9001", 9001)
	require.NoError(t, r.UpsertAggregate(first))

	rows, err := query.Q.RanFeedNotification.WithContext(ctx).
		Where(query.Q.RanFeedNotification.RecipientID.Eq(1001)).Find()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int32(1), rows[0].AggCount)
	assert.Equal(t, int64(2001), rows[0].ActorID)
	assert.Equal(t, int32(0), rows[0].IsRead)
	firstUpdatedAt := rows[0].UpdatedAt

	// 模拟 recipient 标已读
	require.NoError(t, query.Q.RanFeedNotification.WithContext(ctx).UnderlyingDB().
		Exec("UPDATE ran_feed_notification SET is_read=1 WHERE id=?", rows[0].ID).Error)

	// 等 1 毫秒确保 updated_at 前移
	time.Sleep(2 * time.Millisecond)

	// 第二次 upsert 同 (recipient, aggKey) 但换新 actor
	second := mkNotif(1001, 2002, 10, "LF:9001", 9001)
	require.NoError(t, r.UpsertAggregate(second))

	rows, err = query.Q.RanFeedNotification.WithContext(ctx).
		Where(query.Q.RanFeedNotification.RecipientID.Eq(1001)).Find()
	require.NoError(t, err)
	require.Len(t, rows, 1, "同 uk 应仅一行")
	assert.Equal(t, int32(2), rows[0].AggCount, "agg_count 应累加")
	assert.Equal(t, int64(2002), rows[0].ActorID, "actor 应换为最新")
	assert.Equal(t, int32(0), rows[0].IsRead, "re-surface 应重置为未读")
	assert.True(t, rows[0].UpdatedAt.After(firstUpdatedAt), "updated_at 应前移")
}

func TestIntegration_UpsertAggregate_不同AggKey独立(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	require.NoError(t, r.UpsertAggregate(mkNotif(1002, 3001, 10, "LF:100", 100)))
	require.NoError(t, r.UpsertAggregate(mkNotif(1002, 3002, 10, "LF:200", 200)))

	n, err := query.Q.RanFeedNotification.WithContext(ctx).
		Where(query.Q.RanFeedNotification.RecipientID.Eq(1002)).Count()
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestIntegration_Insert_单条(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	row := mkNotif(1003, 4001, 20, "CR:555", 100)
	row.CommentID = 555
	row.Snippet = "回复了你"
	require.NoError(t, r.Insert(row))
	assert.Greater(t, row.ID, int64(0))

	got, err := query.Q.RanFeedNotification.WithContext(ctx).
		Where(query.Q.RanFeedNotification.ID.Eq(row.ID)).First()
	require.NoError(t, err)
	assert.Equal(t, int32(20), got.NotifyType)
	assert.Equal(t, "CR:555", got.AggKey)
	assert.Equal(t, "回复了你", got.Snippet)
}

func TestIntegration_ListByRecipient_复合游标翻页(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	// 造 3 条 同 recipient=1004 updated_at 严格递增 保证顺序稳定
	base := time.Now().Add(-time.Hour)
	for i := int64(1); i <= 3; i++ {
		row := &model.RanFeedNotification{
			RecipientID: 1004,
			ActorID:     5000 + i,
			NotifyType:  10,
			AggKey:      fmt.Sprintf("LF:%d", i),
			AggCount:    1,
			ContentID:   i,
			CreatedAt:   base.Add(time.Duration(i) * time.Second),
			UpdatedAt:   base.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, r.Insert(row))
	}

	// 首页 pageSize=2 应返回 updated_at DESC 前两条 即 aggKey LF:3 LF:2
	page1, err := r.ListByRecipient(1004, 0, time.Time{}, 0, 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.Equal(t, "LF:3", page1[0].AggKey)
	assert.Equal(t, "LF:2", page1[1].AggKey)

	// 用 page1 末条作游标翻页
	cursor := page1[1]
	page2, err := r.ListByRecipient(1004, 0, cursor.UpdatedAt, cursor.ID, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, "LF:1", page2[0].AggKey)

	// 再翻应空
	cursor2 := page2[0]
	page3, err := r.ListByRecipient(1004, 0, cursor2.UpdatedAt, cursor2.ID, 2)
	require.NoError(t, err)
	assert.Len(t, page3, 0)
}

func TestIntegration_ListByRecipient_同updatedAt_按ID_DESC_TieBreak(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	// 造 3 条同 recipient=1005 且同 updated_at 保证只有 id 能区分
	same := time.Now().Truncate(time.Millisecond)
	for i := int64(1); i <= 3; i++ {
		row := &model.RanFeedNotification{
			RecipientID: 1005,
			ActorID:     6000 + i,
			NotifyType:  10,
			AggKey:      fmt.Sprintf("LF:%d", i),
			AggCount:    1,
			CreatedAt:   same,
			UpdatedAt:   same,
		}
		require.NoError(t, r.Insert(row))
	}

	// 首页 pageSize=2 按 id DESC 应返回 id 最大的两条
	page1, err := r.ListByRecipient(1005, 0, time.Time{}, 0, 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.True(t, page1[0].ID > page1[1].ID)

	// 复合游标 (same, page1[1].ID) 应只返回 id < page1[1].ID 的第 3 条
	page2, err := r.ListByRecipient(1005, 0, page1[1].UpdatedAt, page1[1].ID, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.True(t, page2[0].ID < page1[1].ID)
}

func TestIntegration_ListByRecipient_软删过滤与TypeFilter(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	// 三条:两个 type=10 一个 type=20 其中一个 type=10 被软删
	rows := []*model.RanFeedNotification{
		{RecipientID: 1006, ActorID: 7001, NotifyType: 10, AggKey: "LF:1", AggCount: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 1006, ActorID: 7002, NotifyType: 10, AggKey: "LF:2", AggCount: 1, IsDeleted: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 1006, ActorID: 7003, NotifyType: 20, AggKey: "CR:1", AggCount: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, row := range rows {
		require.NoError(t, r.Insert(row))
	}

	all, err := r.ListByRecipient(1006, 0, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.Len(t, all, 2, "软删应过滤")

	onlyLike, err := r.ListByRecipient(1006, 10, time.Time{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, onlyLike, 1, "typeFilter 应生效")
	assert.Equal(t, int32(10), onlyLike[0].NotifyType)
}

func TestIntegration_CountUnread(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	rows := []*model.RanFeedNotification{
		{RecipientID: 1007, ActorID: 1, NotifyType: 10, AggKey: "LF:1", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 1007, ActorID: 2, NotifyType: 10, AggKey: "LF:2", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 1007, ActorID: 3, NotifyType: 10, AggKey: "LF:3", AggCount: 1, IsRead: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 1007, ActorID: 4, NotifyType: 10, AggKey: "LF:4", AggCount: 1, IsRead: 0, IsDeleted: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, row := range rows {
		require.NoError(t, r.Insert(row))
	}

	n, err := r.CountUnread(1007)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n, "未读=2 排除已读与软删")
}

func TestIntegration_MarkRead_越权拒(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	victim := &model.RanFeedNotification{RecipientID: 1008, ActorID: 1, NotifyType: 10, AggKey: "LF:1", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, r.Insert(victim))
	own := &model.RanFeedNotification{RecipientID: 1009, ActorID: 2, NotifyType: 10, AggKey: "LF:2", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, r.Insert(own))

	// 攻击者 1009 尝试标 victim(1008)的通知为已读
	affected, err := r.MarkRead(1009, []int64{victim.ID, own.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected, "只翻自己的一条")

	// victim 那条应仍未读
	got, err := query.Q.RanFeedNotification.WithContext(ctx).
		Where(query.Q.RanFeedNotification.ID.Eq(victim.ID)).First()
	require.NoError(t, err)
	assert.Equal(t, int32(0), got.IsRead, "越权应被 recipient where 拒")
}

func TestIntegration_MarkRead_只翻未读(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	unread := &model.RanFeedNotification{RecipientID: 1010, ActorID: 1, NotifyType: 10, AggKey: "LF:1", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	already := &model.RanFeedNotification{RecipientID: 1010, ActorID: 2, NotifyType: 10, AggKey: "LF:2", AggCount: 1, IsRead: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, r.Insert(unread))
	require.NoError(t, r.Insert(already))

	affected, err := r.MarkRead(1010, []int64{unread.ID, already.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected, "只翻未读的一条 已读的不再变更")
}

func TestIntegration_MarkAllRead(t *testing.T) {
	ctx := setupIntegration(t)
	r := newRealRepo(ctx)

	rows := []*model.RanFeedNotification{
		{RecipientID: 1011, ActorID: 1, NotifyType: 10, AggKey: "LF:1", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 1011, ActorID: 2, NotifyType: 10, AggKey: "LF:2", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 1011, ActorID: 3, NotifyType: 10, AggKey: "LF:3", AggCount: 1, IsRead: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{RecipientID: 9999, ActorID: 4, NotifyType: 10, AggKey: "LF:9", AggCount: 1, IsRead: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, row := range rows {
		require.NoError(t, r.Insert(row))
	}

	affected, err := r.MarkAllRead(1011)
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected, "只翻自己两条未读")

	// 别人的仍未读
	other, err := query.Q.RanFeedNotification.WithContext(ctx).
		Where(query.Q.RanFeedNotification.RecipientID.Eq(9999)).First()
	require.NoError(t, err)
	assert.Equal(t, int32(0), other.IsRead)
}