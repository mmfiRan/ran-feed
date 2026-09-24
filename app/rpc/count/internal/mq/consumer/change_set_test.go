package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	countenum "ran-feed/app/rpc/count/internal/common/enums"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

func TestChangeSet_RecordContent(t *testing.T) {
	cs := newChangeSet()
	cs.record(strategy.Update{
		BizType:    countenum.BizTypeLike,
		TargetType: countenum.TargetTypeContent,
		TargetID:   100,
	}, 9)

	assert.Contains(t, cs.counts, countKey{countenum.BizTypeLike, countenum.TargetTypeContent, 100})
	assert.Contains(t, cs.users, int64(9))
	assert.Contains(t, cs.contents, int64(100))
}

func TestChangeSet_RecordUser(t *testing.T) {
	cs := newChangeSet()
	cs.record(strategy.Update{
		BizType:    countenum.BizTypeFollowing,
		TargetType: countenum.TargetTypeUser,
		TargetID:   7,
	}, 0)

	assert.Contains(t, cs.counts, countKey{countenum.BizTypeFollowing, countenum.TargetTypeUser, 7})
	assert.Contains(t, cs.users, int64(7))
	assert.Empty(t, cs.contents)
}

func TestChangeSet_RecordContentWithoutOwner(t *testing.T) {
	cs := newChangeSet()
	cs.record(strategy.Update{
		BizType:    countenum.BizTypeLike,
		TargetType: countenum.TargetTypeContent,
		TargetID:   100,
	}, 0)

	assert.Empty(t, cs.users)
	assert.Contains(t, cs.contents, int64(100))
}

func TestChangeSet_Empty(t *testing.T) {
	cs := newChangeSet()
	assert.True(t, cs.empty())
	cs.record(strategy.Update{TargetType: countenum.TargetTypeUser, TargetID: 1}, 0)
	assert.False(t, cs.empty())
}

func TestHotDirtyShard(t *testing.T) {
	assert.Equal(t, 0, hotDirtyShard(0))
	assert.Equal(t, 0, hotDirtyShard(-1))
	assert.GreaterOrEqual(t, hotDirtyShard(123), 0)
}
