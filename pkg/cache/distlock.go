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

// ErrLockBusy 等待重试用尽仍未拿到锁/未发现别人重建好结果时返回
// 调用方可据此提示用户稍后重试 或自行决定降级策略
var ErrLockBusy = errors.New("cache: rebuild lock busy after retries")

// DistLocker 分布式锁防击穿器
// 同 lockKey 全集群同时刻只允许一个调用者执行 rebuild
// 未抢到锁的调用者轮询 checkCache 等待别人重建好 超时返回 ErrLockBusy
//
// 适用场景 跨实例共享的热点 key（全局热榜 / 全网热门内容详情 / 全局配置）
// 不适用 per-user 类 key（用户独立 用进程内 Group 已足够）
//
// 注意 锁过期由底层 RedisLock 的 TTL 保证 假定项目已有 watchdog 续期机制
// 本实现不内置续期 长时间重建超过 TTL 时存在双重建可能 调用方需保证 rebuild 在 TTL 内完成
type DistLocker struct {
	store        *redis.Redis
	lockTTL      time.Duration
	waitTimeout  time.Duration
	pollInterval time.Duration
}

// DistLockerOption 配置选项 不传则使用默认值
type DistLockerOption func(*DistLocker)

// WithLockTTL 自定义锁 TTL 默认 30s
func WithLockTTL(d time.Duration) DistLockerOption {
	return func(dl *DistLocker) {
		dl.lockTTL = d
	}
}

// WithWaitTimeout 自定义未抢到锁时的总等待上限 默认 3s
func WithWaitTimeout(d time.Duration) DistLockerOption {
	return func(dl *DistLocker) {
		dl.waitTimeout = d
	}
}

// WithPollInterval 自定义未抢到锁时的轮询间隔 默认 100ms
func WithPollInterval(d time.Duration) DistLockerOption {
	return func(dl *DistLocker) {
		dl.pollInterval = d
	}
}

// NewDistLocker 构造 业务侧通常按"资源类型"持有一个
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

// BuildLockKey 约定的锁 key 命名 业务侧统一用 lock:rebuild:<resourceKey> 避免重名
func BuildLockKey(resourceKey string) string {
	return "lock:rebuild:" + resourceKey
}

// DoWithLock 分布式防击穿入口
//
// 参数
//   - ctx: 调用方 ctx 用于取消等待
//   - lockKey: 互斥锁的 Redis key 推荐用 BuildLockKey 统一命名
//   - checkCache: 拿不到锁时的双检函数 返回 (值, 是否就绪, err)
//     拿到锁后会先执行一次双检 避免重复重建
//   - rebuild: 真正的重建逻辑 在持有锁期间执行
//
// 行为
//
//	尝试拿锁 成功 → 双检 checkCache 未就绪则 rebuild → 释放锁
//	未拿到锁 → 立刻+定时轮询 checkCache 等别人重建好
//	等待超过 waitTimeout 返回 ErrLockBusy
//	ctx 取消立即返回 ctx.Err()
//
// 锁释放使用独立 ctx 防止请求 ctx 提前取消导致锁无法释放
func DoWithLock[T any](
	d *DistLocker,
	ctx context.Context,
	lockKey string,
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
		// 双检 别的实例可能刚重建好才释放锁
		if v, ready, cerr := checkCache(ctx); cerr == nil && ready {
			return v, nil
		}
		return rebuild(ctx)
	}

	return waitForRebuild(ctx, d, checkCache)
}

// waitForRebuild 未抢到锁时的等待路径 立即+定时轮询直到 ready / 超时 / ctx 取消
func waitForRebuild[T any](
	ctx context.Context,
	d *DistLocker,
	checkCache func(ctx context.Context) (T, bool, error),
) (T, error) {
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

// releaseLockSafe 释放锁失败不影响业务 锁会自然 TTL 过期
// 用 background ctx 避免请求 ctx 提前取消导致锁无法释放
func releaseLockSafe(lock *redis.RedisLock, lockKey string) {
	bg, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := lock.ReleaseCtx(bg); err != nil {
		logx.Errorf("release dist lock failed: key=%s err=%v", lockKey, err)
	}
}
