---
name: go-coding
description: 编写或修改本项目 Go 代码时遵循的编码规范 — 命名 错误处理 三层架构 新增接口(goctl 生成 API/RPC) proto 编写规范 枚举创建与使用 并发与 context 缓存 cache-aside 配置新增 注释与日志风格 文件组织 设计模式选择。在动手写 logic repository handler proto api、创建或修改枚举、加缓存/配置、引入新抽象前应用；只要碰 Go 生产代码就应先看本规范。
---

# ran-feed Go 编码规范

编写本项目 Go 代码时遵循。先看本文；要判断"是否该引入某个设计模式"时再读
[references/design-patterns.md](references/design-patterns.md)。

## 命名

遵循 [Effective Go](https://go.dev/doc/effective_go) 与 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)：

- **包名**：全小写单词，不用下划线，与目录名一致（`userservice`、`feedservice`）
- **接口名**：单方法接口以动词+`er` 结尾（`Stringer`、`Reader`）；多方法接口描述能力（`Repository`、`Strategy`）
- **错误变量**：哨兵错误以 `Err` 开头（`ErrNotFound`）；自定义错误类型以 `Error` 结尾
- **缩写词保持大小写一致**：`userID` / `UserID`，不写 `userId` / `UserId`
- **避免无意义名称**：`data`、`info`、`result` 需改为描述性名称（`contentRow`、`likeEvent`）

## 错误处理

```go
// 业务错误用 errorx.NewMsg 附带可读信息
return nil, errorx.NewMsg("用户不存在")

// 包装底层错误 保留调用链 用于非预期错误
return nil, errorx.Wrap(ctx, err, errorx.NewMsg("查询用户失败"))

// 不要吞掉错误 也不要只 log 不返回
```

- RPC 返回值在使用前**必须检查 nil**：`if resp == nil || resp.UserInfo == nil { ... }`
- 错误信息用中文面向用户，英文面向开发者日志
- gRPC handler 内不直接 `panic`，由 `ServerGrpcInterceptor` 统一恢复

## 项目分层模式

严格遵循三层结构，**不可越层调用**：

```
Handler / Server  →  Logic  →  Repository
                  ↘          ↗
                   svc.ServiceContext（依赖容器）
```

- **Logic 层**：业务编排，调用 Repository 和外部 RPC，不直接写 SQL
- **Repository 层**：只做数据读写，不含业务判断，返回 `*model.Xxx` 或原始错误
- **Handler/Server 层**：只做参数绑定与响应序列化，不含业务逻辑

## 新增接口（API / RPC）

新增/修改对外接口**先改定义文件再用 goctl 生成，绝不手改生成产物**；API 与 RPC 生成都带 `--style=go_zero`，
改了 `.api` 还要重生成 swagger。命令 格式约束（`XxxReq`/`XxxRes` 字段风格 swagger 步骤）见
[references/new-interface.md](references/new-interface.md)。改 `.proto` 属升级处理，先与用户确认。

**HTTP 接口遵循 RESTful 风格**：

- 写操作（增删改）用 POST / PUT / DELETE，**GET 不产生副作用**
- 例外：**条件查询参数很多（超过 3 个查询条件）时**，可以用 POST 表示查询（避免超长 query string 与可读性差）
- 判断依据：改数据的接口必须是非 GET；只有纯读且条件少的查询才用 GET

## proto 编写规范

新增或修改 `.proto`（`app/rpc/**/proto/*.proto`、`pkg/commonpb/common.proto`）时先读
[references/proto-style.md](references/proto-style.md)：package/go_package 前缀 枚举命名与 UNSPECIFIED
Timestamp 时间字段 Empty 空响应 定义顺序 reserved total 注释 以及 Go 侧常量命名联动。

## 并发与 context

```go
// 并行 RPC 调用用 mapreduce / errgroup
mr.Finish(func() error { ... }, func() error { ... })

// 共享状态用 sync.Mutex 保护 锁粒度尽量小
// goroutine 必须有明确退出条件和 context 取消响应
// 不要用 time.Sleep 轮询 用 channel 或 context.Done
```

- 启动 goroutine 前明确"谁负责回收"；用 go-zero 的 `threading.GoSafe` 代替裸 `go func()`
- **ctx 永远是第一个参数**：`func(ctx context.Context, ...)`；不把 ctx 存进结构体，不传 `nil`
- **异步副作用脱离请求 ctx**：请求返回后请求 ctx 会被取消，fire-and-forget 的副作用
  （发 MQ 缓存清理 fan-out 收件箱回填）必须换成独立 bg ctx 加显式 timeout，否则会被请求结束打断

```go
// 异步副作用 脱离请求 ctx 加 5s 超时
bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
threading.GoSafe(func() {
    defer cancel()
    // 推送 MQ 或清理缓存
})
```

## 接口与依赖注入

```go
// 依赖接口 便于测试和替换
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*model.RanFeedUser, error)
}

// 不要依赖具体实现类型
```

- `svc.ServiceContext` 是唯一依赖容器，所有外部依赖（DB、Redis、RPC client）通过它传递
- 新增依赖在 `ServiceContext` 中初始化，不在 Logic 内部自行创建

## 缓存规范（cache-aside）

读路径加缓存统一走旁路缓存（参考 `usercache/`）：正负值都缓存加负哨兵防穿透 TTL 叠 jitter 抗雪崩
只降级不阻断 批量一次 RTT 写路径 `Invalidate` 失效。防击穿分层 per-user 用 `cache.Group` 单飞
全局热点用 `cache.DistLocker`。详见 [references/caching.md](references/caching.md)。

## 配置新增

新增配置在 `internal/config` 加字段 用 `json:",default=..."` 给安全默认值 etc yaml 写值
零值能安全关功能 凭据走 env。详见 [references/config.md](references/config.md)。

## 注释

只在**业务流程的关键流转节点**加中文注释。**一条注释精简成一句话说清意图**：

```go
// 标注流程节点

// 不复述代码 不解释参数 不描述显而易见的操作
```

- **能单行就不用多行**：优先 `//` 单行注释，能一句话说清就不拆成多行、不用 `/* */` 块注释
- 一句话写不下时，先想是不是注释太啰嗦或函数职责太杂，而不是直接堆多行
- 写了 TODO 就及时登记或解决，不写完就消失

**注释默认不带任何标点符号**：用空格分隔短语，运算符写成中文词。

- 去掉散文标点：`，。、：；（）「」§·` 以及 `= + / *` 这类符号
- 保留代码字面量：redis key（如 `feed:hot:dirty`）函数名 `log10` `TopN` `ZADD` 这类 token 照写
- 运算符用中文词代替：等于 加 减 乘 除以

```go
// 算分 log10 加权互动 加 发布秒 除以 S
// ZADD 覆盖非 ZINCRBY 分值是按总量重算的时点值 自愈漂移
```

**日志通俗易懂且精简**：go-zero 的 `logx` 已做结构化处理（`level` `content` `caller` 与链路
`trace` `span` 自动带上），业务日志正常用即可，不必额外套字段 API。一句话说清什么操作出了什么问题，
不堆术语不带长串符号。

- 始终带 ctx：用 `logx.WithContext(ctx).Xxx` 或 `logc.Xxx(ctx, ...)`，trace/span 才会串联链路
- 关键变量用 `key=value` 附在句尾，便于阅读与排查

```go
// 好
l.Errorf("热榜快照切换失败 snapshotID=%s err=%v", id, err)

// 差
l.Errorf("RebuildHotSnapshotScript EvalCtx 执行异常, 详见: %+v", err)
```

## 文件组织

- 单文件不超过 **400 行**；超出说明职责不单一，考虑拆分
- 魔法值不在代码里内联，提取为具名常量（放置分级见下节「常量与枚举」）

## 常量与枚举

**常量按作用域分级放置，不在 logic / handler / repository 里内联散落 `const`**：

- **全局通用** → `pkg/consts/`（普通常量）。判定：被 ≥2 服务共用，或属项目级通用约定（超时、软删标记）
- **单服务专属** → 该服务 `internal/common/consts/xxx_consts.go`，按主题分文件（`content_consts.go`、`redis/redis_consts.go`）

**枚举（DB 枚举字段）** 的创建与使用有独立规范，见 [references/enum.md](references/enum.md)。
核心原则：DB 持久化字段不自造字面量，复用业务枚举或 pb 枚举；从 DB 行转枚举用
`content.ContentStatus(row.Status)`，需拦非法值用 `Parse` 校验 `Valid`。

## 设计模式

总原则：**先有清晰需求再选模式，不为"优雅"而过度设计**。三个相似实现是抽象的起点，
一个或两个不是。本项目常用 8 种模式（策略 / 工厂 / 仓储 / 中间件 / 函数式选项 /
发布订阅 / 适配器 / 模板方法）的触发场景、写法、项目示例与反例，见
[references/design-patterns.md](references/design-patterns.md)。**只在有明确需求时引入新模式，先复用再新增。**
