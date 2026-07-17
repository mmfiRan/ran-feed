package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/notification/internal/entity/model"
)

// 项目条件下无 sqlmock/sqlite Repo 层 SQL 语义靠 gorm-gen + feat-notify-007 端到端联调
// 本测只覆盖参数守卫(recipient<=0/agg_key 空/ids 空/limit<=0)保证脏输入不误入 DB
// 守卫早于任何 gorm 调用 故 db=nil tx=nil 不 panic

func newRepoWithoutDB() *notificationRepositoryImpl {
	return &notificationRepositoryImpl{ctx: context.Background()}
}

func TestUpsertAggregate_Guards(t *testing.T) {
	r := newRepoWithoutDB()
	assert.NoError(t, r.UpsertAggregate(nil))
	assert.NoError(t, r.UpsertAggregate(&model.RanFeedNotification{RecipientID: 0, AggKey: "LF:1"}))
	assert.NoError(t, r.UpsertAggregate(&model.RanFeedNotification{RecipientID: -1, AggKey: "LF:1"}))
	assert.NoError(t, r.UpsertAggregate(&model.RanFeedNotification{RecipientID: 1, AggKey: ""}))
}

func TestInsert_Guards(t *testing.T) {
	r := newRepoWithoutDB()
	assert.NoError(t, r.Insert(nil))
	assert.NoError(t, r.Insert(&model.RanFeedNotification{RecipientID: 0, AggKey: "CR:1"}))
	assert.NoError(t, r.Insert(&model.RanFeedNotification{RecipientID: 1, AggKey: ""}))
}

func TestListByRecipient_Guards(t *testing.T) {
	r := newRepoWithoutDB()

	rows, err := r.ListByRecipient(0, 0, time.Time{}, 0, 10)
	assert.NoError(t, err)
	assert.Nil(t, rows)

	rows, err = r.ListByRecipient(-1, 0, time.Time{}, 0, 10)
	assert.NoError(t, err)
	assert.Nil(t, rows)

	rows, err = r.ListByRecipient(1, 0, time.Time{}, 0, 0)
	assert.NoError(t, err)
	assert.Nil(t, rows)

	rows, err = r.ListByRecipient(1, 0, time.Time{}, 0, -5)
	assert.NoError(t, err)
	assert.Nil(t, rows)
}

func TestCountUnread_Guards(t *testing.T) {
	r := newRepoWithoutDB()
	n, err := r.CountUnread(0)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)

	n, err = r.CountUnread(-1)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestMarkRead_Guards(t *testing.T) {
	r := newRepoWithoutDB()

	n, err := r.MarkRead(0, []int64{1, 2})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)

	n, err = r.MarkRead(1, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)

	n, err = r.MarkRead(1, []int64{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestMarkAllRead_Guards(t *testing.T) {
	r := newRepoWithoutDB()
	n, err := r.MarkAllRead(0)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)

	n, err = r.MarkAllRead(-100)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestDedup_Guards(t *testing.T) {
	r := &mqConsumeDedupRepositoryImpl{ctx: context.Background()}
	ok, err := r.InsertIfAbsent("", "evt-1")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = r.InsertIfAbsent("notification.canal_consumer", "")
	assert.NoError(t, err)
	assert.False(t, ok)
}