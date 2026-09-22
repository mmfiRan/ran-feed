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

func TestHotDirtyShard(t *testing.T) {
	assert.Equal(t, 0, hotDirtyShard(0))
	assert.Equal(t, 0, hotDirtyShard(-1))
	assert.GreaterOrEqual(t, hotDirtyShard(123), 0)
}
