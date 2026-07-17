package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"
)

// ---------- mocks ----------

type mockDedupRepo struct {
	seen  map[string]bool
	calls int
}

func (m *mockDedupRepo) WithTx(*query.Query) repositories.MqConsumeDedupRepository { return m }

func (m *mockDedupRepo) InsertIfAbsent(_, eventID string) (bool, error) {
	m.calls++
	if m.seen[eventID] {
		return false, nil
	}
	m.seen[eventID] = true
	return true, nil
}

type mockNotifyRepo struct {
	upsertCalls   int
	insertCalls   int
	unread        int64
	lastUpserted  *model.RanFeedNotification
	lastInserted  *model.RanFeedNotification
	markReadCalls int
}

func (m *mockNotifyRepo) WithTx(*query.Query) repositories.NotificationRepository { return m }
func (m *mockNotifyRepo) UpsertAggregate(row *model.RanFeedNotification) error {
	m.upsertCalls++
	m.lastUpserted = row
	return nil
}
func (m *mockNotifyRepo) Insert(row *model.RanFeedNotification) error {
	m.insertCalls++
	m.lastInserted = row
	return nil
}
func (m *mockNotifyRepo) ListByRecipient(int64, int32, time.Time, int64, int) ([]*model.RanFeedNotification, error) {
	return nil, nil
}
func (m *mockNotifyRepo) CountUnread(int64) (int64, error) { return m.unread, nil }
func (m *mockNotifyRepo) MarkRead(int64, []int64) (int64, error) {
	m.markReadCalls++
	return 0, nil
}
func (m *mockNotifyRepo) MarkAllRead(int64) (int64, error) { return 0, nil }

// stubStrategy 由测试注入 便于精确控制 ExtractEvents 输出
type stubStrategy struct {
	table  string
	events []strategy.NotifyEvent
	calls  int
}

func (s *stubStrategy) TableName() string { return s.table }
func (s *stubStrategy) ExtractEvents(context.Context, string, map[string]interface{}, map[string]interface{}) []strategy.NotifyEvent {
	s.calls++
	return s.events
}

// newTestConsumer 构造能在无 DB tx 下跑的 consumer(所有 mock 的 WithTx 忽略参数)
func newTestConsumer(notifyRepo repositories.NotificationRepository, dedupRepo repositories.MqConsumeDedupRepository) *CanalNotificationConsumer {
	ctx := context.Background()
	return &CanalNotificationConsumer{
		ctx:        ctx,
		svcContext: &svc.ServiceContext{},
		Logger:     logx.WithContext(ctx),
		notifyRepo: notifyRepo,
		dedupRepo:  dedupRepo,
		strategies: nil,
	}
}

// ---------- persistEvent 分派 ----------

func TestPersistEvent_Aggregate_调UpsertAggregate(t *testing.T) {
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify, &mockDedupRepo{seen: map[string]bool{}})
	err := c.persistEvent(notify, strategy.NotifyEvent{
		RecipientID: 200, ActorID: 100,
		NotifyType: int32(notification.NotifyType_LIKE_FAVORITE),
		AggKey:     "LF:500", Action: strategy.PersistAggregate, ContentID: 500,
	}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, notify.upsertCalls)
	assert.Equal(t, 0, notify.insertCalls)
	assert.Equal(t, int64(200), notify.lastUpserted.RecipientID)
	assert.Equal(t, "LF:500", notify.lastUpserted.AggKey)
	assert.Equal(t, int32(0), notify.lastUpserted.IsRead)
	assert.Equal(t, int64(100), notify.lastUpserted.CreatedBy, "created_by 记 actor")
}

func TestPersistEvent_InsertOne_调Insert(t *testing.T) {
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify, &mockDedupRepo{seen: map[string]bool{}})
	err := c.persistEvent(notify, strategy.NotifyEvent{
		RecipientID: 200, ActorID: 100,
		NotifyType: int32(notification.NotifyType_COMMENT_REPLY),
		AggKey:     "CR:1000", Action: strategy.PersistInsertOne,
		ContentID: 500, CommentID: 1000, Snippet: "hi",
	}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0, notify.upsertCalls)
	assert.Equal(t, 1, notify.insertCalls)
	assert.Equal(t, "hi", notify.lastInserted.Snippet)
	assert.Equal(t, int64(1000), notify.lastInserted.CommentID)
}

func TestPersistEvent_updatedAt零值取当前(t *testing.T) {
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify, &mockDedupRepo{seen: map[string]bool{}})
	require.NoError(t, c.persistEvent(notify, strategy.NotifyEvent{
		RecipientID: 1, ActorID: 2, Action: strategy.PersistAggregate, AggKey: "LF:1",
	}, time.Time{}))
	assert.False(t, notify.lastUpserted.UpdatedAt.IsZero(), "零 updated_at 应替换为 time.Now")
}

// ---------- processRow 幂等 + 落库 ----------

func TestProcessRow_Dedup命中跳过不调Strategy(t *testing.T) {
	dedup := &mockDedupRepo{seen: map[string]bool{}}
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify, dedup)

	stub := &stubStrategy{table: "ran_feed_like", events: []strategy.NotifyEvent{
		{RecipientID: 200, ActorID: 100, Action: strategy.PersistAggregate, AggKey: "LF:1"},
	}}
	meta := rowMeta{table: stub.table, op: "INSERT", eventID: "evt", updatedAt: time.Now(), strategy: stub}
	row := map[string]interface{}{"id": int64(1)}

	// 第 1 次 落库
	recipients, err := c.processRow(context.Background(), nil, meta, 0, row, nil)
	require.NoError(t, err)
	assert.Equal(t, []int64{200}, recipients)
	assert.Equal(t, 1, stub.calls)
	assert.Equal(t, 1, notify.upsertCalls)

	// 第 2 次 dedup 命中 strategy 与 notify 均不再被调
	recipients, err = c.processRow(context.Background(), nil, meta, 0, row, nil)
	require.NoError(t, err)
	assert.Nil(t, recipients)
	assert.Equal(t, 1, stub.calls, "dedup 命中 不再调 strategy")
	assert.Equal(t, 1, notify.upsertCalls, "幂等重投不重复落库")
	assert.Equal(t, 2, dedup.calls, "dedup 每次都查")
}

func TestProcessRow_Strategy空事件不调Notify(t *testing.T) {
	dedup := &mockDedupRepo{seen: map[string]bool{}}
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify, dedup)

	stub := &stubStrategy{table: "ran_feed_like", events: nil} // 例:自我过滤/DELETE
	meta := rowMeta{table: stub.table, op: "INSERT", eventID: "evt", updatedAt: time.Now(), strategy: stub}

	recipients, err := c.processRow(context.Background(), nil, meta, 0, map[string]interface{}{"id": int64(1)}, nil)
	require.NoError(t, err)
	assert.Empty(t, recipients)
	assert.Equal(t, 0, notify.upsertCalls)
	assert.Equal(t, 0, notify.insertCalls)
}

func TestProcessRow_多事件全部落库(t *testing.T) {
	dedup := &mockDedupRepo{seen: map[string]bool{}}
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify, dedup)

	// 单行产两个事件(边界场景 目前 strategy 不产 但接口允许)
	stub := &stubStrategy{table: "ran_feed_comment", events: []strategy.NotifyEvent{
		{RecipientID: 200, ActorID: 100, Action: strategy.PersistInsertOne, AggKey: "CR:1"},
		{RecipientID: 300, ActorID: 100, Action: strategy.PersistInsertOne, AggKey: "CR:2"},
	}}
	meta := rowMeta{table: stub.table, op: "INSERT", eventID: "evt", updatedAt: time.Now(), strategy: stub}

	recipients, err := c.processRow(context.Background(), nil, meta, 0, map[string]interface{}{"id": int64(1)}, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{200, 300}, recipients)
	assert.Equal(t, 2, notify.insertCalls)
}
