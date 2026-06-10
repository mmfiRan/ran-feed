// Package cache 项目级缓存基础设施 提供两套防击穿抽象
//
//   - Group      进程内单飞 解决"同 pod 内并发去重" 适合 per-user 类 key
//   - DistLocker 分布式锁   解决"跨实例集群去重"   适合全局共享热点 key
//
// 两者可以组合 单 pod 内先用 Group 合并到 1 个再去抢分布式锁 当前未做组合
package cache

import "golang.org/x/sync/singleflight"

// Group 单飞分组 同一 key 同时刻只允许一个 fn 实际执行 其余调用者等待并复用结果
// 内部封装 golang.org/x/sync/singleflight 提供泛型友好的 Do 函数
type Group struct {
	sf singleflight.Group
}

// NewGroup 构造一个独立的分组 业务侧通常按"资源类型"持有一个
func NewGroup() *Group {
	return &Group{}
}

// Do 在 key 维度上单飞执行 fn
// shared=true 表示本次返回值同时被其他等待者复用 用于监控热点 key
//
// 注意事项
//   - fn 内部应使用独立 ctx 避免请求 ctx 取消串扰其他等待者
//   - fn 内部应自行 defer recover 把 panic 转 error 否则会传播到所有等待者
//   - fn 失败结果会广播给当前等待者 inflight 自然结束后下一波请求可正常重试
func Do[T any](g *Group, key string, fn func() (T, error)) (value T, err error, shared bool) {
	raw, err, shared := g.sf.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		var zero T
		return zero, err, shared
	}
	return raw.(T), nil, shared
}

// Forget 主动清除 key 的飞行记录 不等待当前调用返回
// 默认无需调用 仅用于"重建失败后允许立即重试"等特殊场景
func (g *Group) Forget(key string) {
	g.sf.Forget(key)
}
