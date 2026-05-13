# 工作规则与约束

## 工作规则

- **每次只做一个功能**：从 `feature_list.json` 选取**一个** `not-started` 或 `in-progress` 的条目
- **验证后才算完成**：未通过 `./init.sh` 不得声称功能完成
- **不扩大范围**：只修改与当前功能直接相关的文件，不做顺手重构
- **会话结束前更新状态**：更新 `progress.md` 和 `feature_list.json`，记录完成证据
- **保持可重启状态**：下一个会话必须能立即运行 `./init.sh`

## 关键约束（违反会引入 Bug）

- **`*.gen.go` 不可手动修改**：`internal/entity/model/*.gen.go`、`internal/entity/query/*.gen.go` 均为 GORM Gen 生成产物
- **`routes.go` 不可手动修改**：`app/front/internal/handler/routes.go` 由 goctl 生成，只改 `.api` 文件后重新生成
- **不在代码中硬编码凭据**：本地开发在根目录放 `.env`（参考 `deploy/.env`）
- **Redis 多步操作用 Lua 脚本**：参考 `internal/common/utils/lua/` 中的脚本模式,但是如果能够在一个redis指令中完成的话尽量采用gozero提供的redis命令即可
- **软删除必须过滤**：所有 Repository 查询必须加 `IsDeleted.Eq(0)`

## 完成标准

一个功能**仅在满足以下全部条件时**才能标记为 `done`：

- [ ] 目标行为已实现
- [ ] `./init.sh` 验证通过（`go build ./...` + `go vet ./...`）
- [ ] 完成证据已记录在 `feature_list.json` 的 `evidence` 字段
- [ ] 仓库可重启（下一会话能直接运行 `./init.sh`）

## 会话结束前

1. 更新 `progress.md`（当前状态、决策、遗留风险）
2. 更新 `feature_list.json`（新状态 + 完成证据）
3. 提交：`git add -p && git commit -m "feat/fix: <描述>"`
4. 再跑一次 `./init.sh` 确认干净

## 升级处理

遇到以下情况时暂停并询问用户：

- **架构决策**：新增服务、修改 proto 接口、改变数据库 Schema
- **安全相关**：凭据处理、认证逻辑变更
- **依赖变更**：修改 `go.mod` 引入新库
- **需求不明确**：`feature_list.json` 中描述有歧义

---

## Git 提交规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/)，与 `feature_list.json` 联动。

### 提交格式

```
<type>(<scope>): <subject>

[可选 body：解释 why，引用 feature_list ID]

[可选 footer：BREAKING CHANGE 或 issue 引用]
```

### Type（必填，二选一不能合并）

| Type | 用途 | 示例 |
|------|------|------|
| `feat` | 新功能（对应 feature_list 的 feat-XXX） | `feat(feed): 实现推拉结合 fan-out` |
| `fix` | Bug 修复（对应 fix-XXX / sec-XXX） | `fix(user): 登录禁用账号校验` |
| `refactor` | 重构（不改变可观察行为） | `refactor(content): 提取 afterPublish` |
| `perf` | 性能优化 | `perf(comment): 评论列表加 Redis 缓存` |
| `docs` | 仅文档变更 | `docs: 完善 rules.md 设计模式枚举` |
| `test` | 测试相关 | `test(jwt): 补充 token 解析单测` |
| `chore` | 杂项（依赖、构建脚本、配置） | `chore: 升级 go-zero 到 v1.7` |
| `style` | 仅格式变更（不改逻辑） | `style: gofmt 全量格式化` |
| `ci` | CI/CD 配置变更 | `ci: 增加 vet 检查步骤` |

### Scope（推荐）

用服务名（`user` / `content` / `interaction` / `count` / `front`）或模块名（`feed` / `auth` / `oss` / `comment`），让 `git log --grep="(feed)"` 这类检索有效。

### Subject（必填）

- 中英文都可以，但**同一仓库保持一致**（本项目用中文）
- ≤ 50 字符
- 动词开头，不加句号
- 描述"做了什么"，不重复 Type 信息

### Body（推荐）

- 解释 **为什么**（diff 已经显示了 what）
- 每行 ≤ 72 字符
- 涉及 `feature_list.json` 条目时 **必须** 引用 ID

示例：

```
fix(content): Redis 操作移出 MySQL 事务

事务回调内 EvalCtx 失败会触发 MySQL 回滚但 Redis 已写入，
违反 rules.md 的"事务内不调用外部 RPC 或 Redis"约束。
改为 commit 后 best-effort 更新，缓存失败由懒加载兜底。

相关：fix-011
```

### 反例

```text
❌ "update"                              # 无信息
❌ "feat: add some changes"              # 模糊
❌ "fix bug"                             # 没说什么 bug
❌ "feat: fix-010 + fix-011 + feat-019"  # 多条目混合提交
❌ "Updated XXX (no test, no review)"    # 工作日志而非变更说明
```

### 与流程联动

- **一次提交对应一个 feature_list 条目**，与"每次只做一个功能"对齐
- **提交前必须 `./init.sh` 通过**，编译/vet 失败不准 commit
- Body 引用条目 ID，形成 commit ↔ feature ↔ progress.md 的可追溯链
- 永远不用 `--no-verify` 跳过 git hooks，hook 报错先修问题

---

## 编码规范

### 命名

遵循 [Effective Go](https://go.dev/doc/effective_go) 与 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)：

- **包名**：全小写单词，不用下划线，与目录名一致（`userservice`、`feedservice`）
- **接口名**：单方法接口以动词+`er` 结尾（`Stringer`、`Reader`）；多方法接口描述能力（`Repository`、`Strategy`）
- **错误变量**：哨兵错误以 `Err` 开头（`ErrNotFound`）；自定义错误类型以 `Error` 结尾
- **缩写词保持大小写一致**：`userID` / `UserID`，不写 `userId` / `UserId`
- **避免无意义名称**：`data`、`info`、`result` 需改为描述性名称（`contentRow`、`likeEvent`）

### 错误处理

```go
// ✅ 业务错误用 errorx.NewMsg，附带可读信息
return nil, errorx.NewMsg("用户不存在")

// ✅ 包装底层错误，保留调用链（用于非预期错误）
return nil, errorx.Wrap(ctx, err, errorx.NewMsg("查询用户失败"))

// ❌ 不要吞掉错误，也不要只 log 不返回
_ = someOperation()
logx.Error(err) // 然后继续执行，调用方不知道出错了
```

- RPC 返回值在使用前**必须检查 nil**：`if resp == nil || resp.UserInfo == nil { ... }`
- 错误信息用中文面向用户，英文面向开发者日志
- gRPC handler 内不直接 `panic`，由 `ServerGrpcInterceptor` 统一恢复

### 项目分层模式

本项目严格遵循三层结构，**不可越层调用**：

```
Handler / Server  →  Logic  →  Repository
                  ↘          ↗
                   svc.ServiceContext（依赖容器）
```

- **Logic 层**：业务编排，调用 Repository 和外部 RPC，不直接写 SQL
- **Repository 层**：只做数据读写，不含业务判断，返回 `*model.Xxx` 或原始错误
- **Handler/Server 层**：只做参数绑定与响应序列化，不含业务逻辑

### 设计模式使用原则

**总原则**：先有清晰需求再选模式，不为"优雅"而过度设计。三个相似实现是抽象的起点，一个或两个不是。下面列出本项目常用 / 适用的模式与触发场景。

#### 1. 策略模式（Strategy）

- **触发场景**：同一抽象行为有多种可替换实现，且实现会随业务扩展（多云存储、多类型 MQ 事件、多种推送渠道）
- **写法**：定义 `XxxStrategy` 接口；用 `Registry` 或 `Factory` 注册；调用方只依赖接口
- **项目示例**：`count-rpc` 的 `internal/mq/consumer/strategy/`、`content-rpc` 的 `internal/common/oss/`
- **反例**：只有一种实现且未来无扩展需求时不要预先抽象

#### 2. 工厂模式（Factory）

- **触发场景**：创建逻辑复杂（多参数、需要根据配置选择实现）
- **写法**：`NewXxxFactory()` + `factory.Get(name)` / `factory.MustGet(name)`
- **项目示例**：`content-rpc` 的 `internal/common/oss/factory.go`
- **反例**：简单类型直接 `NewXxx()` 即可，不要为构造而构造

#### 3. 仓储模式（Repository）

- **触发场景**：Logic 层访问持久化数据（MySQL、Redis）
- **写法**：`XxxRepository` 接口 + impl 结构体；impl 通过 `getQuery()` + `WithTx(tx)` 支持事务上下文切换
- **项目示例**：`*/internal/repositories/*.go`
- **必须遵守**：
  - `ServiceContext` 中只暴露接口类型，不暴露 impl
  - 软删除查询必须 `IsDeleted.Eq(0)`
  - 跨 Repository 事务用 `query.Q.Transaction(func(tx *query.Query) error { ... })` + `repo.WithTx(tx)` 串联

#### 4. 装饰器 / 中间件（Middleware / Interceptor）

- **触发场景**：横切关注点（日志、鉴权、限流、追踪、panic 恢复）叠加到核心流程
- **写法**：
  - HTTP：`func(http.HandlerFunc) http.HandlerFunc`
  - gRPC：`grpc.UnaryServerInterceptor` / `grpc.UnaryClientInterceptor`
- **项目示例**：`pkg/interceptor/interceptor.go`、`app/front/internal/middleware/*.go`
- **反例**：业务专属逻辑不是横切关注点，不要塞到中间件里

#### 5. 函数式选项（Functional Options）

- **触发场景**：可选参数 ≥ 3 个，且参数会随时间膨胀；构造函数签名需要长期演化
- **写法**：`type Option func(*config)` + `WithXxx(v) Option` + `NewXxx(required, opts ...Option)`
- **反例**：参数稳定且大部分必填时用结构体直接传即可

#### 6. 发布订阅 / 事件驱动（Pub-Sub）

- **触发场景**：跨服务异步通信、最终一致性（计数维护、索引同步、收件箱回填）
- **写法**：Producer 只关心"发出什么事件"；Consumer 用策略模式 dispatch 不同事件类型
- **项目示例**：Canal binlog → Kafka → `count-rpc` consumer
- **反例**：强一致性场景（支付、库存扣减）不要异步化

#### 7. 适配器模式（Adapter）

- **触发场景**：第三方 SDK 接口与项目抽象不一致，或要屏蔽 SDK 升级带来的影响
- **写法**：用项目自己的接口包装 SDK，业务代码只依赖接口
- **项目示例**：`oss/strategy/aliyun_strategy.go` 包装阿里云 SDK
- **反例**：SDK 接口已经够干净时不要为加层而加层

#### 8. 模板方法（Template Method）

- **触发场景**：多个流程骨架相同、只有少数步骤不同（如 hot_fast / hot_cold 这类同结构 job）
- **写法**：父类型 / 上层函数定义骨架，把可变部分提为接口或 hook 函数注入
- **反例**：只有两个实现且骨架差异不大时直接平铺两份代码更清晰

**只在有明确需求时引入新模式**。先复用现有模式，再考虑新增。

### 并发

```go
// ✅ 并行 RPC 调用用 mapreduce / errgroup
mr.Finish(func() error { ... }, func() error { ... })

// ✅ 共享状态用 sync.Mutex 保护，锁的粒度尽量小
// ✅ goroutine 必须有明确的退出条件和 context 取消响应
// ❌ 不要用 time.Sleep 轮询，用 channel 或 context.Done()
```

- 启动 goroutine 前明确"谁负责回收"；使用 `go-zero` 的 `threading.GoSafe` 代替裸 `go func()`

### 接口与依赖注入

```go
// ✅ 依赖接口，便于测试和替换
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*model.RanFeedUser, error)
}

// ❌ 不要依赖具体实现类型
type Logic struct {
    repo *userRepositoryImpl  // 写死了实现
}
```

- `svc.ServiceContext` 是唯一的依赖容器，所有外部依赖（DB、Redis、RPC client）通过它传递
- 新增依赖在 `ServiceContext` 中初始化，不在 Logic 内部自行创建

### 注释

只在**业务流程的关键流转节点**加中文注释，一句话，精简：

```go
// ✅ 标注流程节点
// 1. 写库
// 2. 异步推送热榜
// 3. 回填关注收件箱

// ❌ 复述代码、解释参数、描述显而易见的操作
// 获取用户信息
user, err := repo.GetByID(ctx, id)
```

- 不写 TODO 后就消失；写了 TODO 就在 `feature_list.json` 中登记

### 其他

- 单文件不超过 **400 行**；超出说明职责不单一，考虑拆分
- 魔法数字提取为具名常量，放到 `internal/common/consts/` 下对应文件
- 数据库事务内不调用外部 RPC 或 Redis；副作用操作放事务提交后执行