package consumer

import (
	"context"
	"testing"
	"time"

	"ran-feed/pkg/event/canal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOutboxEventKey_PrefersOutboxEventID 幂等键优先取 outbox event_id 雪花号 与对账作业同值域
func TestOutboxEventKey_PrefersOutboxEventID(t *testing.T) {
	row := map[string]any{"event_id": "1900000000000000001", "id": "42"}
	assert.Equal(t, "1900000000000000001", outboxEventKey("evt-1", outboxTable, opInsert, row, 0))
}

// TestOutboxEventKey_FallsBackToCanalKey 缺 event_id 时退回 canal 批次派生键 保证老消息仍幂等
func TestOutboxEventKey_FallsBackToCanalKey(t *testing.T) {
	row := map[string]any{"id": "42"}
	want := canal.RowEventID("evt-1", outboxTable, opInsert, row, 0)
	require.NotEmpty(t, want)
	assert.Equal(t, want, outboxEventKey("evt-1", outboxTable, opInsert, row, 0))
}

// TestRetryWithBackoff_RecoversAfterTransient 瞬时失败在重试窗口内成功 返回 nil 且不超次数
func TestRetryWithBackoff_RecoversAfterTransient(t *testing.T) {
	calls := 0
	err := retryWithBackoff(context.Background(), 3, time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return assert.AnError
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

// TestRetryWithBackoff_ExhaustsReturnsLastErr 一直失败则用满次数并返回最后错误
func TestRetryWithBackoff_ExhaustsReturnsLastErr(t *testing.T) {
	calls := 0
	err := retryWithBackoff(context.Background(), 3, time.Millisecond, func() error {
		calls++
		return assert.AnError
	})

	require.Error(t, err)
	assert.Equal(t, 3, calls, "应恰好重试 attempts 次")
}

// TestRetryWithBackoff_CtxCancelStops 退避等待期间 ctx 取消应立即返回 不耗满次数
func TestRetryWithBackoff_CtxCancelStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := retryWithBackoff(ctx, 5, 50*time.Millisecond, func() error {
		calls++
		cancel()
		return assert.AnError
	})

	require.Error(t, err)
	assert.Equal(t, 1, calls, "ctx 取消后不应继续重试")
}
