package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

func TestChangeSet_RecordContent(t *testing.T) {
	cs := newChangeSet()
	cs.record(strategy.Update{
		BizType:    count.BizType_BIZ_TYPE_LIKE,
		TargetType: count.TargetType_TARGET_TYPE_CONTENT,
		TargetID:   100,
	}, 9)

	assert.Contains(t, cs.counts, countKey{count.BizType_BIZ_TYPE_LIKE, count.TargetType_TARGET_TYPE_CONTENT, 100})
	assert.Contains(t, cs.users, int64(9))
	assert.Contains(t, cs.contents, int64(100))
}

func TestChangeSet_RecordUser(t *testing.T) {
	cs := newChangeSet()
	cs.record(strategy.Update{
		BizType:    count.BizType_BIZ_TYPE_FOLLOWING,
		TargetType: count.TargetType_TARGET_TYPE_USER,
		TargetID:   7,
	}, 0)

	assert.Contains(t, cs.counts, countKey{count.BizType_BIZ_TYPE_FOLLOWING, count.TargetType_TARGET_TYPE_USER, 7})
	assert.Contains(t, cs.users, int64(7))
	assert.Empty(t, cs.contents)
}

func TestChangeSet_RecordContentWithoutOwner(t *testing.T) {
	cs := newChangeSet()
	cs.record(strategy.Update{
		BizType:    count.BizType_BIZ_TYPE_LIKE,
		TargetType: count.TargetType_TARGET_TYPE_CONTENT,
		TargetID:   100,
	}, 0)

	assert.Empty(t, cs.users)
	assert.Contains(t, cs.contents, int64(100))
}

func TestChangeSet_Empty(t *testing.T) {
	cs := newChangeSet()
	assert.True(t, cs.empty())
	cs.record(strategy.Update{TargetType: count.TargetType_TARGET_TYPE_USER, TargetID: 1}, 0)
	assert.False(t, cs.empty())
}

func TestCanalMessage_Helpers(t *testing.T) {
	msg := canalMessage{Table: "  Ran_Feed_Like ", Type: "insert ", Ts: 1700000000}
	assert.Equal(t, "ran_feed_like", msg.table())
	assert.Equal(t, "INSERT", msg.op())
	assert.Equal(t, int64(1700000000), msg.updatedAt().Unix())

	milli := canalMessage{Ts: 1700000000000}
	assert.Equal(t, int64(1700000000), milli.updatedAt().Unix())

	zero := canalMessage{Ts: 0}
	assert.False(t, zero.updatedAt().IsZero())
}

func TestCanalMessage_EventID(t *testing.T) {
	assert.Equal(t, "evt-1", canalMessage{ID: "evt-1"}.eventID("raw"))

	long := canalMessage{ID: string(make([]byte, 100))}
	assert.Len(t, long.eventID("raw"), 64)

	// 无 id 时对原始报文取 sha1 长度 40
	assert.Len(t, canalMessage{ID: nil}.eventID("raw"), 40)
}

func TestCanalMessage_OldRow(t *testing.T) {
	msg := canalMessage{Old: []map[string]interface{}{{"status": 10}}}
	assert.Equal(t, 10, msg.oldRow(0)["status"])
	assert.Nil(t, msg.oldRow(1))
	assert.Nil(t, msg.oldRow(-1))
}

func TestRowEventID_Deterministic(t *testing.T) {
	row := map[string]interface{}{"id": int64(5)}
	a := rowEventID("evt", "ran_feed_like", "INSERT", row, 0)
	b := rowEventID("evt", "ran_feed_like", "INSERT", row, 0)
	assert.Equal(t, a, b)

	other := rowEventID("evt", "ran_feed_like", "INSERT", map[string]interface{}{"id": int64(6)}, 0)
	assert.NotEqual(t, a, other)
}

func TestHotDirtyShard(t *testing.T) {
	assert.Equal(t, 0, hotDirtyShard(0))
	assert.Equal(t, 0, hotDirtyShard(-1))
	assert.GreaterOrEqual(t, hotDirtyShard(123), 0)
}
