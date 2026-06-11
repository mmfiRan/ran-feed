# 设计模式使用原则

**总原则**：先有清晰需求再选模式，不为"优雅"而过度设计。三个相似实现是抽象的起点，
一个或两个不是。下面列出本项目常用 / 适用的模式与触发场景。

## 1. 策略模式（Strategy）

- **触发场景**：同一抽象行为有多种可替换实现，且实现会随业务扩展（多云存储、多类型 MQ 事件、多种推送渠道）
- **写法**：定义 `XxxStrategy` 接口；用 `Registry` 或 `Factory` 注册；调用方只依赖接口
- **项目示例**：`count-rpc` 的 `internal/mq/consumer/strategy/`、`content-rpc` 的 `internal/common/oss/`
- **反例**：只有一种实现且未来无扩展需求时不要预先抽象

## 2. 工厂模式（Factory）

- **触发场景**：创建逻辑复杂（多参数、需要根据配置选择实现）
- **写法**：`NewXxxFactory()` + `factory.Get(name)` / `factory.MustGet(name)`
- **项目示例**：`content-rpc` 的 `internal/common/oss/factory.go`
- **反例**：简单类型直接 `NewXxx()` 即可，不要为构造而构造

## 3. 仓储模式（Repository）

- **触发场景**：Logic 层访问持久化数据（MySQL、Redis）
- **写法**：`XxxRepository` 接口 + impl 结构体；impl 通过 `getQuery()` + `WithTx(tx)` 支持事务上下文切换
- **项目示例**：`*/internal/repositories/*.go`
- **必须遵守**：
  - `ServiceContext` 中只暴露接口类型，不暴露 impl
  - 软删除查询必须 `IsDeleted.Eq(0)`
  - 跨 Repository 事务用 `query.Q.Transaction(func(tx *query.Query) error { ... })` + `repo.WithTx(tx)` 串联

## 4. 装饰器 / 中间件（Middleware / Interceptor）

- **触发场景**：横切关注点（日志、鉴权、限流、追踪、panic 恢复）叠加到核心流程
- **写法**：
  - HTTP：`func(http.HandlerFunc) http.HandlerFunc`
  - gRPC：`grpc.UnaryServerInterceptor` / `grpc.UnaryClientInterceptor`
- **项目示例**：`pkg/interceptor/interceptor.go`、`app/front/internal/middleware/*.go`
- **反例**：业务专属逻辑不是横切关注点，不要塞到中间件里

## 5. 函数式选项（Functional Options）

- **触发场景**：可选参数 ≥ 3 个，且参数会随时间膨胀；构造函数签名需要长期演化
- **写法**：`type Option func(*config)` + `WithXxx(v) Option` + `NewXxx(required, opts ...Option)`
- **反例**：参数稳定且大部分必填时用结构体直接传即可

## 6. 发布订阅 / 事件驱动（Pub-Sub）

- **触发场景**：跨服务异步通信、最终一致性（计数维护、索引同步、收件箱回填）
- **写法**：Producer 只关心"发出什么事件"；Consumer 用策略模式 dispatch 不同事件类型
- **项目示例**：Canal binlog → Kafka → `count-rpc` consumer
- **反例**：强一致性场景（支付、库存扣减）不要异步化

## 7. 适配器模式（Adapter）

- **触发场景**：第三方 SDK 接口与项目抽象不一致，或要屏蔽 SDK 升级带来的影响
- **写法**：用项目自己的接口包装 SDK，业务代码只依赖接口
- **项目示例**：`oss/strategy/aliyun_strategy.go` 包装阿里云 SDK
- **反例**：SDK 接口已经够干净时不要为加层而加层

## 8. 模板方法（Template Method）

- **触发场景**：多个流程骨架相同、只有少数步骤不同（如 hot_fast / hot_cold 这类同结构 job）
- **写法**：父类型 / 上层函数定义骨架，把可变部分提为接口或 hook 函数注入
- **反例**：只有两个实现且骨架差异不大时直接平铺两份代码更清晰

---

**只在有明确需求时引入新模式**。先复用现有模式，再考虑新增。
