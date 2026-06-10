package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func newTestLocker(t *testing.T, opts ...DistLockerOption) (*DistLocker, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	return NewDistLocker(r, opts...), mr
}

// TestDistLock_AcquireAndRebuild 抢到锁直接执行 rebuild 释放锁
func TestDistLock_AcquireAndRebuild(t *testing.T) {
	dl, mr := newTestLocker(t)
	ctx := context.Background()

	v, err := DoWithLock(dl, ctx, BuildLockKey("k"),
		func(_ context.Context) (string, bool, error) { return "", false, nil },
		func(_ context.Context) (string, error) { return "rebuilt", nil },
	)
	require.NoError(t, err)
	assert.Equal(t, "rebuilt", v)
	assert.False(t, mr.Exists(BuildLockKey("k")), "重建完成后锁应被释放")
}

// TestDistLock_DoubleCheckSkipRebuild 双检发现 cache 已就绪时 rebuild 不应执行
func TestDistLock_DoubleCheckSkipRebuild(t *testing.T) {
	dl, _ := newTestLocker(t)
	var rebuilt int32

	v, err := DoWithLock(dl, context.Background(), BuildLockKey("k"),
		func(_ context.Context) (string, bool, error) { return "cached", true, nil },
		func(_ context.Context) (string, error) {
			atomic.AddInt32(&rebuilt, 1)
			return "rebuilt", nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "cached", v, "应返回 cache 命中值")
	assert.Equal(t, int32(0), atomic.LoadInt32(&rebuilt), "双检命中 rebuild 不应执行")
}

// TestDistLock_ConcurrentDedup 同 lockKey 并发只有一个 goroutine 执行 rebuild
// 其余轮询 cache 等结果
func TestDistLock_ConcurrentDedup(t *testing.T) {
	dl, _ := newTestLocker(t,
		WithWaitTimeout(2*time.Second),
		WithPollInterval(20*time.Millisecond),
	)

	var rebuildCount int32
	var sharedCache atomic.Value // 模拟集群共享缓存

	check := func(_ context.Context) (string, bool, error) {
		v := sharedCache.Load()
		if v == nil {
			return "", false, nil
		}
		return v.(string), true, nil
	}
	rebuild := func(_ context.Context) (string, error) {
		atomic.AddInt32(&rebuildCount, 1)
		time.Sleep(100 * time.Millisecond) // 模拟重建耗时
		sharedCache.Store("rebuilt")
		return "rebuilt", nil
	}

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	start := make(chan struct{})
	results := make([]string, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		idx := i
		go func() {
			defer wg.Done()
			<-start
			v, err := DoWithLock(dl, context.Background(), BuildLockKey("k"), check, rebuild)
			results[idx] = v
			errs[idx] = err
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&rebuildCount), "全集群 rebuild 应只执行一次")
	for i, err := range errs {
		require.NoError(t, err, "goroutine %d", i)
		assert.Equal(t, "rebuilt", results[i])
	}
}

// TestDistLock_WaitTimeoutReturnsErrLockBusy 拿不到锁且别人迟迟不重建 超时返回 ErrLockBusy
func TestDistLock_WaitTimeoutReturnsErrLockBusy(t *testing.T) {
	dl, mr := newTestLocker(t,
		WithLockTTL(30*time.Second),
		WithWaitTimeout(200*time.Millisecond),
		WithPollInterval(30*time.Millisecond),
	)

	// 提前在 Redis 写入锁 让本次调用拿不到
	mr.Set(BuildLockKey("k"), "other-owner")
	mr.SetTTL(BuildLockKey("k"), 30*time.Second)

	_, err := DoWithLock(dl, context.Background(), BuildLockKey("k"),
		func(_ context.Context) (string, bool, error) { return "", false, nil },
		func(_ context.Context) (string, error) {
			t.Fatal("rebuild 不应被本调用执行")
			return "", nil
		},
	)
	assert.ErrorIs(t, err, ErrLockBusy)
}

// TestDistLock_WaitCachePopulatedDuringPoll 拿不到锁但等待期间 cache 就绪 应返回缓存值
func TestDistLock_WaitCachePopulatedDuringPoll(t *testing.T) {
	dl, mr := newTestLocker(t,
		WithWaitTimeout(2*time.Second),
		WithPollInterval(30*time.Millisecond),
	)
	mr.Set(BuildLockKey("k"), "other-owner")

	var ready atomic.Bool
	// 100ms 后模拟别的实例重建完
	go func() {
		time.Sleep(100 * time.Millisecond)
		ready.Store(true)
	}()

	v, err := DoWithLock(dl, context.Background(), BuildLockKey("k"),
		func(_ context.Context) (string, bool, error) {
			if ready.Load() {
				return "from-other-instance", true, nil
			}
			return "", false, nil
		},
		func(_ context.Context) (string, error) {
			t.Fatal("rebuild 不应被本调用执行")
			return "", nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "from-other-instance", v)
}

// TestDistLock_CtxCancelReturnsImmediately ctx 取消时立即返回不等超时
func TestDistLock_CtxCancelReturnsImmediately(t *testing.T) {
	dl, mr := newTestLocker(t,
		WithWaitTimeout(5*time.Second),
		WithPollInterval(50*time.Millisecond),
	)
	mr.Set(BuildLockKey("k"), "other-owner")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := DoWithLock(dl, ctx, BuildLockKey("k"),
		func(_ context.Context) (string, bool, error) { return "", false, nil },
		func(_ context.Context) (string, error) { return "x", nil },
	)
	elapsed := time.Since(start)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Less(t, elapsed, 500*time.Millisecond, "ctx 取消后应快速返回 而非等满 waitTimeout")
}

// TestDistLock_RebuildErrorReleasesLock rebuild 报错时锁仍被正常释放
func TestDistLock_RebuildErrorReleasesLock(t *testing.T) {
	dl, mr := newTestLocker(t)
	wantErr := errors.New("rebuild boom")

	_, err := DoWithLock(dl, context.Background(), BuildLockKey("k"),
		func(_ context.Context) (string, bool, error) { return "", false, nil },
		func(_ context.Context) (string, error) { return "", wantErr },
	)
	assert.ErrorIs(t, err, wantErr)
	assert.False(t, mr.Exists(BuildLockKey("k")), "重建报错也应释放锁")
}

// TestDistLock_BuildLockKey 锁 key 命名约定
func TestDistLock_BuildLockKey(t *testing.T) {
	assert.Equal(t, "lock:rebuild:like:user:1", BuildLockKey("like:user:1"))
}
