package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	notifyenum "ran-feed/app/rpc/notification/internal/common/enums"
	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/pkg/event/pipeline"
)

// ---------- mocks ----------

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
func (m *mockNotifyRepo) UpsertReview(row *model.RanFeedNotification) error {
	m.upsertCalls++
	m.lastUpserted = row
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
func newTestConsumer(notifyRepo repositories.NotificationRepository) *CanalNotificationConsumer {
	ctx := context.Background()
	return &CanalNotificationConsumer{
		ctx:        ctx,
		svcContext: &svc.ServiceContext{},
		Logger:     logx.WithContext(ctx),
		notifyRepo: notifyRepo,
		strategies: nil,
	}
}

// newTestMeta 行元信息 去重键由管道算 此处只需业务字段
func newTestMeta(table, op string) pipeline.RowMeta {
	return pipeline.RowMeta{Table: table, Op: op, EventID: "evt", UpdatedAt: time.Now()}
}

// ---------- persistEvent 分派 ----------

func TestPersistEvent_Aggregate_调UpsertAggregate(t *testing.T) {
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify)
	err := c.persistEvent(notify, strategy.NotifyEvent{
		RecipientID: 200, ActorID: 100,
		NotifyType: notifyenum.NotifyTypeLikeFavorite,
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
	c := newTestConsumer(notify)
	err := c.persistEvent(notify, strategy.NotifyEvent{
		RecipientID: 200, ActorID: 100,
		NotifyType: notifyenum.NotifyTypeCommentReply,
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
	c := newTestConsumer(notify)
	require.NoError(t, c.persistEvent(notify, strategy.NotifyEvent{
		RecipientID: 1, ActorID: 2, Action: strategy.PersistAggregate, AggKey: "LF:1",
	}, time.Time{}))
	assert.False(t, notify.lastUpserted.UpdatedAt.IsZero(), "零 updated_at 应替换为 time.Now")
}

// ---------- processRow 落库 去重由管道负责 ----------

func TestProcessRow_事件落库并返回recipient(t *testing.T) {
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify)

	stub := &stubStrategy{table: "ran_feed_like", events: []strategy.NotifyEvent{
		{RecipientID: 200, ActorID: 100, Action: strategy.PersistAggregate, AggKey: "LF:1"},
	}}
	row := map[string]any{"id": int64(1)}

	recipients, err := c.processRow(context.Background(), nil, newTestMeta(stub.table, "INSERT"), stub, row, nil)
	require.NoError(t, err)
	assert.Equal(t, []int64{200}, recipients)
	assert.Equal(t, 1, stub.calls)
	assert.Equal(t, 1, notify.upsertCalls)
}

func TestProcessRow_Strategy空事件不调Notify(t *testing.T) {
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify)

	stub := &stubStrategy{table: "ran_feed_like", events: nil} // 例:自我过滤/DELETE

	recipients, err := c.processRow(context.Background(), nil, newTestMeta(stub.table, "INSERT"), stub, map[string]any{"id": int64(1)}, nil)
	require.NoError(t, err)
	assert.Empty(t, recipients)
	assert.Equal(t, 0, notify.upsertCalls)
	assert.Equal(t, 0, notify.insertCalls)
}

func TestProcessRow_多事件全部落库(t *testing.T) {
	notify := &mockNotifyRepo{}
	c := newTestConsumer(notify)

	// 单行产两个事件(边界场景 目前 strategy 不产 但接口允许)
	stub := &stubStrategy{table: "ran_feed_comment", events: []strategy.NotifyEvent{
		{RecipientID: 200, ActorID: 100, Action: strategy.PersistInsertOne, AggKey: "CR:1"},
		{RecipientID: 300, ActorID: 100, Action: strategy.PersistInsertOne, AggKey: "CR:2"},
	}}

	recipients, err := c.processRow(context.Background(), nil, newTestMeta(stub.table, "INSERT"), stub, map[string]any{"id": int64(1)}, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{200, 300}, recipients)
	assert.Equal(t, 2, notify.insertCalls)
}
