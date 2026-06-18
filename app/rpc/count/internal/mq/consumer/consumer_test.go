package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/entity/model"
	"ran-feed/app/rpc/count/internal/entity/query"
	counterservicelogic "ran-feed/app/rpc/count/internal/logic/counterservice"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"
)

func newRegistry(t *testing.T) *strategy.Registry {
	t.Helper()
	return strategy.NewDefaultRegistry()
}

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

type mockCountRepo struct {
	getResult                 *model.RanFeedCountValue
	updateDeltaCalls          int
	updateDeltaWithOwnerCalls int
}

func (m *mockCountRepo) WithTx(*query.Query) repositories.CountValueRepository { return m }

func (m *mockCountRepo) Get(int32, int32, int64) (*model.RanFeedCountValue, error) {
	return m.getResult, nil
}

func (m *mockCountRepo) BatchGet(int32, int32, []int64) (map[int64]*model.RanFeedCountValue, error) {
	return nil, nil
}

func (m *mockCountRepo) SumByOwner(int32, int32, int64) (int64, error) { return 0, nil }
func (m *mockCountRepo) ListTargetValuesByValueGte(int32, int32, int64) ([]*model.RanFeedCountValue, error) {
	return nil, nil
}

func (m *mockCountRepo) UpsertValue(int32, int32, int64, int64, time.Time) error { return nil }

func (m *mockCountRepo) UpdateDelta(int32, int32, int64, int64, time.Time) (int64, error) {
	m.updateDeltaCalls++
	return 0, nil
}

func (m *mockCountRepo) UpdateDeltaWithOwner(int32, int32, int64, int64, int64, time.Time) (int64, error) {
	m.updateDeltaWithOwnerCalls++
	return 0, nil
}

func newTestConsumer(t *testing.T, countRepo repositories.CountValueRepository, dedupRepo repositories.MqConsumeDedupRepository) *CanalCountConsumer {
	t.Helper()
	ctx := context.Background()
	return &CanalCountConsumer{
		ctx:           ctx,
		svcContext:    &svc.ServiceContext{},
		Logger:        logx.WithContext(ctx),
		countRepo:     countRepo,
		dedupRepo:     dedupRepo,
		deltaOperator: counterservicelogic.NewCountDeltaOperator(ctx, &svc.ServiceContext{}),
		consumerName:  "test",
		strategies:    nil,
	}
}

func TestProcessRow_DedupSkipsSecondTime(t *testing.T) {
	ctx := context.Background()
	dedup := &mockDedupRepo{seen: map[string]bool{}}
	countRepo := &mockCountRepo{}
	c := newTestConsumer(t, countRepo, dedup)

	s, ok := newRegistry(t).Get("ran_feed_like")
	require.True(t, ok)
	meta := rowMeta{table: "ran_feed_like", op: "INSERT", eventID: "evt", updatedAt: time.Now(), strategy: s}
	row := map[string]interface{}{"content_id": int64(100), "content_user_id": int64(9), "status": 10}
	cs := newChangeSet()

	require.NoError(t, c.processRow(ctx, nil, meta, 0, row, nil, cs))
	require.NoError(t, c.processRow(ctx, nil, meta, 0, row, nil, cs))

	assert.Equal(t, 2, dedup.calls)
	assert.Equal(t, 1, countRepo.updateDeltaWithOwnerCalls)
	assert.Contains(t, cs.counts, countKey{count.BizType_LIKE, count.TargetType_CONTENT, 100})
}

func TestProcessRow_ResetToZeroCascades(t *testing.T) {
	ctx := context.Background()
	dedup := &mockDedupRepo{seen: map[string]bool{}}
	countRepo := &mockCountRepo{getResult: &model.RanFeedCountValue{Value: 5, OwnerID: 9}}
	c := newTestConsumer(t, countRepo, dedup)

	s, ok := newRegistry(t).Get("ran_feed_content")
	require.True(t, ok)
	meta := rowMeta{table: "ran_feed_content", op: "UPDATE", eventID: "evt", updatedAt: time.Now(), strategy: s}
	row := map[string]interface{}{"id": int64(100), "user_id": int64(9), "is_deleted": 1}
	oldRow := map[string]interface{}{"is_deleted": 0}
	cs := newChangeSet()

	require.NoError(t, c.processRow(ctx, nil, meta, 0, row, oldRow, cs))

	assert.Equal(t, 3, countRepo.updateDeltaWithOwnerCalls)
	assert.Contains(t, cs.contents, int64(100))
	assert.Contains(t, cs.users, int64(9))
}

func TestProcessRow_ResetToZeroSkipsWhenAlreadyZero(t *testing.T) {
	ctx := context.Background()
	dedup := &mockDedupRepo{seen: map[string]bool{}}
	countRepo := &mockCountRepo{getResult: &model.RanFeedCountValue{Value: 0}}
	c := newTestConsumer(t, countRepo, dedup)

	s, ok := newRegistry(t).Get("ran_feed_content")
	require.True(t, ok)
	meta := rowMeta{table: "ran_feed_content", op: "UPDATE", eventID: "evt", updatedAt: time.Now(), strategy: s}
	row := map[string]interface{}{"id": int64(100), "user_id": int64(9), "is_deleted": 1}
	oldRow := map[string]interface{}{"is_deleted": 0}
	cs := newChangeSet()

	require.NoError(t, c.processRow(ctx, nil, meta, 0, row, oldRow, cs))

	assert.Equal(t, 0, countRepo.updateDeltaWithOwnerCalls)
	assert.True(t, cs.empty())
}

func TestRegistry_AllTablesRegistered(t *testing.T) {
	reg := newRegistry(t)
	for _, table := range []string{"ran_feed_like", "ran_feed_favorite", "ran_feed_comment", "ran_feed_follow", "ran_feed_content"} {
		_, ok := reg.Get(table)
		assert.True(t, ok, "未注册 %s", table)
	}
}
