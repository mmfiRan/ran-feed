// Package cache
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	defaultLockTTL      = 30 * time.Second
	defaultWaitTimeout  = 3 * time.Second
	defaultPollInterval = 100 * time.Millisecond
)

var ErrLockBusy = errors.New("缓存重建繁忙稍后再试")

// DistLocker 跨实例防击穿 同一 lockKey 只放一个调用者执行 rebuild
// 未抢到锁的轮询 checkCache 等别人重建好 超时返回 ErrLockBusy
type DistLocker struct {
	store        *redis.Redis
	lockTTL      time.Duration
	waitTimeout  time.Duration
	pollInterval time.Duration
}

type DistLockerOption func(*DistLocker)

// WithLockTTL 锁 TTL 默认 30s
func WithLockTTL(d time.Duration) DistLockerOption {
	return func(dl *DistLocker) {
		dl.lockTTL = d
	}
}

// WithWaitTimeout 未抢到锁时的总等待上限 默认 3s
func WithWaitTimeout(d time.Duration) DistLockerOption {
	return func(dl *DistLocker) {
		dl.waitTimeout = d
	}
}

// WithPollInterval 未抢到锁时的轮询间隔 默认 100ms
func WithPollInterval(d time.Duration) DistLockerOption {
	return func(dl *DistLocker) {
		dl.pollInterval = d
	}
}

func NewDistLocker(store *redis.Redis, opts ...DistLockerOption) *DistLocker {
	dl := &DistLocker{
		store:        store,
		lockTTL:      defaultLockTTL,
		waitTimeout:  defaultWaitTimeout,
		pollInterval: defaultPollInterval,
	}
	for _, opt := range opts {
		opt(dl)
	}
	return dl
}

// BuildLockKey 统一锁 key 命名 lock:rebuild: 加资源 key 避
func BuildLockKey(resourceKey string) string {
	return "lock:rebuild:" + resourceKey
}

// DoWithLock
// 抢到锁 先 checkCache 双检一次 还没就绪才 rebuild 释放锁
// 没抢到 轮询 checkCache 等别人重建好
func DoWithLock[T any](d *DistLocker, ctx context.Context, lockKey string,
	checkCache func(ctx context.Context) (T, bool, error),
	rebuild func(ctx context.Context) (T, error),
) (T, error) {
	var zero T

	lock := redis.NewRedisLock(d.store, lockKey)
	lock.SetExpire(int(d.lockTTL / time.Second))

	acquired, err := lock.AcquireCtx(ctx)
	if err != nil {
		return zero, err
	}

	if acquired {
		defer releaseLockSafe(lock, lockKey)
		if v, ready, cerr := checkCache(ctx); cerr == nil && ready {
			return v, nil
		}
		return rebuild(ctx)
	}

	return waitForRebuild(ctx, d, checkCache)
}

// waitForRebuild 没抢到锁的等待路径 先立即查一次再定时轮询
func waitForRebuild[T any](ctx context.Context, d *DistLocker, checkCache func(ctx context.Context) (T, bool, error)) (T, error) {
	var zero T

	if v, ready, err := checkCache(ctx); err == nil && ready {
		return v, nil
	}

	deadline := time.Now().Add(d.waitTimeout)
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-ticker.C:
			if v, ready, err := checkCache(ctx); err == nil && ready {
				return v, nil
			}
			if time.Now().After(deadline) {
				return zero, ErrLockBusy
			}
		}
	}
}

func releaseLockSafe(lock *redis.RedisLock, lockKey string) {
	bg, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := lock.ReleaseCtx(bg); err != nil {
		logx.Errorf("释放redis锁失败: key=%s err=%v", lockKey, err)
	}
}
