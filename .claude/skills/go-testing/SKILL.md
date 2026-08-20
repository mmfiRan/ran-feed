---
name: go-testing
description: 为本项目编写 Go 单元测试时遵循的规范 — testify 断言 miniredis 模拟 Redis 接口 mock 表驱动用例 确定性隔离。在新增/修改测试或为功能补测试时应用。
---

# ran-feed Go 测试规范

参考 `app/rpc/user/internal/common/utils/usercache/cache_test.go`。提交前 `go test ./...`
必须全绿。

## 测试栈

- **断言**：`github.com/stretchr/testify`
  - **前置条件 / 致命错误用 `require`**：失败即终止当前用例（如 `require.NoError(t, err)` 拿不到依赖就没必要继续）
  - **结果断言用 `assert`**：失败记录但继续，一次跑出多个问题
- **Redis 依赖用 miniredis**：`github.com/alicebob/miniredis/v2` 起内存 Redis，配 `redis.MustNewRedis`
- **DB / 仓储用手写 mock**：实现对应 Repository 接口，不连真库

## 表驱动

```go
tests := []struct {
    name string
    // 输入与期望
}{
    {name: "命中正值"},
    {name: "命中负哨兵"},
    {name: "miss 回源"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

## 测试环境与清理

- 测试与被测**同包**（white-box），可直接验证未导出函数
- setup 抽成 helper 并标 `t.Helper()`；资源释放用 `t.Cleanup`（如 `t.Cleanup(mr.Close)`）

```go
func newTestEnv(t *testing.T) (*redis.Redis, *miniredis.Miniredis, config.UserCacheConfig) {
    t.Helper()
    mr, err := miniredis.Run()
    require.NoError(t, err)
    t.Cleanup(mr.Close)
    ...
}
```

## 确定性隔离

- **关掉不确定因素再断言**：测 TTL 时把 jitter 设 0（`JitterMaxSeconds: 0`），才能断言确定值
- **用调用计数验证缓存行为**：mock 里累加 `getByIDCalls` 之类计数，断言"第二次查走缓存没回源 DB"
- 不依赖真实时间 / 网络 / 随机；miniredis 可用 `mr.FastForward` 推进过期

## 命名

- 文件 `xxx_test.go`，函数 `TestXxx`，子用例名用 `t.Run` 描述场景
- mock 类型命名 `mockXxx`，只实现被测路径需要的方法，其余返回零值占位