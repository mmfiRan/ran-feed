package hot_cold_update

import (
	"context"
	"testing"
	"time"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// TestDeriveWindowDays 验证冷更窗口从半衰期推导 与竞争视界对齐
// 半衰期非正退化兜底 默认 24h 约等于现状 调大半衰期窗口跟随 封顶 maxWindowDays 防全表扫
func TestDeriveWindowDays(t *testing.T) {
	cases := []struct {
		halfLifeHours float64
		want          int
	}{
		{0, defaultWindowDays},  // 非正退化兜底
		{-5, defaultWindowDays}, // 非正退化兜底
		{24, 17},                // 默认 约等于原 15d 现状
		{48, 34},                // 半衰期翻倍 窗口跟随放大
		{72, 50},                // 线性放大
		{168, maxWindowDays},    // 7天半衰期 推导超上限 封顶 90
		{720, maxWindowDays},    // 30天半衰期 仍封顶 防全表扫
	}
	for _, c := range cases {
		got := deriveWindowDays(c.halfLifeHours)
		assert.Equalf(t, c.want, got, "deriveWindowDays(%.0f)", c.halfLifeHours)
	}
}

// TestColdPendingFlagLifecycle 验证冷更预约标志的置位 探测 摘牌
// 快更靠 ExistsCtx 探测此标志决定是否让路 标志带 TTL 防异常未清永久挡快更
func TestColdPendingFlagLifecycle(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	ctx := context.Background()
	key := rediskey.RedisFeedHotColdPendingKey

	// 初始无标志 快更不让路
	exists, err := r.ExistsCtx(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)

	// 冷更挂牌 带 TTL
	require.NoError(t, r.SetexCtx(ctx, key, "1", 3600))
	exists, err = r.ExistsCtx(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists, "挂牌后快更应探测到标志并让路")

	// TTL 已设置 防异常未清永久挡快更
	ttl := mr.TTL(key)
	assert.Greater(t, ttl, time.Duration(0), "预约标志必须带 TTL")

	// 冷更收尾摘牌
	_, err = r.DelCtx(ctx, key)
	require.NoError(t, err)
	exists, err = r.ExistsCtx(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists, "摘牌后快更恢复正常抢锁")
}

// TestAcquireWriteLockBounded_FreeLockAcquiresImmediately
// 写锁空闲时 有界轮询应立即拿到 不进入等待
func TestAcquireWriteLockBounded_FreeLockAcquiresImmediately(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	j := &HotColdUpdateJob{}
	lock := redis.NewRedisLock(r, rediskey.BuildHotFeedWriteLockKey())
	lock.SetExpire(3600)

	start := time.Now()
	locked, err := j.acquireWriteLockBounded(context.Background(), lock)
	require.NoError(t, err)
	assert.True(t, locked, "锁空闲应立即拿到")
	assert.Less(t, time.Since(start), time.Duration(defaultAcquireRetryInterval)*time.Second,
		"空闲锁不应进入轮询等待")
}

// TestAcquireWriteLockBounded_CtxCancelReturnsPromptly
// 写锁被占 ctx 取消时应及时返回 不死等到超时 不拖垮任务
func TestAcquireWriteLockBounded_CtxCancelReturnsPromptly(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})

	// 先占住写锁 模拟一轮快更正在跑
	holder := redis.NewRedisLock(r, rediskey.BuildHotFeedWriteLockKey())
	holder.SetExpire(3600)
	held, err := holder.AcquireCtx(context.Background())
	require.NoError(t, err)
	require.True(t, held)

	j := &HotColdUpdateJob{}
	lock := redis.NewRedisLock(r, rediskey.BuildHotFeedWriteLockKey())
	lock.SetExpire(3600)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	locked, err := j.acquireWriteLockBounded(ctx, lock)
	assert.Error(t, err, "ctx 取消应返回 err")
	assert.False(t, locked)
	assert.Less(t, time.Since(start), time.Duration(defaultAcquireWaitSeconds)*time.Second,
		"ctx 取消应及时返回 不死等到 60s 超时")
}
