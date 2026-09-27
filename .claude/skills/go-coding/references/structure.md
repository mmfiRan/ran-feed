# 结构规范

跨用例的目录、依赖方向与契约归属。**新增目录、新增组件、新增跨服务常量、拆文件之前读本文。**

机检：`bash .claude/skills/go-coding/scripts/check_structure.sh`，退出码 0 通过、1 有违规并逐条打印文件行号。
`.harness/init.sh` 的第 5 步就是调它。写完自查比等提交时被拦便宜。

可读版本见 [`docs/architecture.md`](../../../../docs/architecture.md) 的「结构规范」章——那份是给人看的，
本文补的是规则背后的**理由**与容易搞错的地方。

## 分层与依赖方向

```
Handler / Server → Logic → Repository 接口
                      ↘         ↘
                    Component    Entity / Query / Model
```

依赖只能向下。几条容易踩的：

- **Logic 只引 `internal/repositories`（接口包），不引 `internal/repositories/<impl>`。** 引了实现子包就等于
  绕过接口直接绑死实现，测试也没法换替身。接口和实现分家就是为了这条边可替换。
- **Repository 不引 Logic 或 Component。** 它是最底层，往上引会成环。
- **`pkg/*` 不反向引 `app/*`。** `pkg` 是被各服务共用的，反过来依赖某个服务的话，那个服务就再也拆不出去了。
  编译器能挡跨服务的 `internal` import，挡不住 `pkg → app`，所以靠机检。
- **Handler 只做参数绑定与响应序列化**，有 `if` 判断业务就该下沉。

## 三个入口点互不调用

一个 RPC 服务有三个入口：**RPC Handler、MQ Consumer、Cron Job**。它们互相禁止调用——不是洁癖，是因为
三者的事务边界、重试语义、ctx 生命周期都不一样（Consumer 会重投、Cron 没有请求 ctx、Handler 有请求 ctx）。
互相调用会把三种语义搅在一起，出问题时无法判断是哪条链路的责任。

于是产生一个必须回答的问题：**被两个以上入口点需要的能力放哪？** 放进 `internal/common/component/`。
这也是 Component 存在的唯一理由——它是三个入口点之间唯一合法的共享落点。

- Consumer 与 Cron 的 import 列表里不出现 `logic/`，也不互相引用
- **Consumer 不引业务 Repository**：它只该做协议解析、幂等、重试、调用具名 Component。
  直接查库意味着业务判断漏进了消费链路，重投时会重复执行
- **Cron 与 Consumer 不保存 `ctx` 字段**：ctx 是每次调用的，存进结构体就成了跨请求复用，
  取消与超时会串台（`go-coding` 主文件里也说了 ctx 永远是第一个参数）
- **Job 接显式依赖，不接 `svcCtx`**：接整个 ServiceContext 会让 Job 能碰到任意依赖，
  依赖关系从签名上就看不出来了
- **Logic 只保显式依赖，不存完整 `ServiceContext`**：同上，构造函数仍可以收 svcCtx 做组装，
  但结构体字段只留它真正用的那几个

## Component 契约

Component 是「有依赖 有构造 无请求状态 参与事务但不拥有事务」的能力。四条硬规矩：

- 只接**自己的** Config 子结构，不接整个 ServiceContext
- 不保存请求 ctx
- 不偷偷起 goroutine（起了就得有人负责回收，调用方看不见）
- 不负责事务边界、不拥有事务；返回 Model DO 或自己的结果类型，**不直接返回 protobuf**

最后一条的用意：Component 是领域能力，不是传输层。返回 pb 会让它绑死在某个 RPC 契约上，
另一个入口点想复用就得被迫接受同样的契约。

### 内部按角色切文件 不按技术切

一个 Component 的文件划分跟着**能力内的角色**走，不跟着技术走：

```text
component/hotfeed/
├── feed.go       对外能力入口 (New / Rebuild / UpdateIncremental)
├── options.go    参数与归一化
├── score.go      算分
├── rebuild.go    全量重建
├── incremental.go 增量
├── snapshot.go   快照
└── lua.go + scripts/   Lua 与其归属表
```

反面是 `types.go` / `utils.go` / `constants.go` 这种按技术切的切法——它把「一个能力」拆散到四个文件里，
读代码的人要跳四次才能拼出一个完整流程。

**`.lua` 与它的 `//go:embed` 变量必须同包**，所以脚本跟着组件走：`.lua` 放组件的 `scripts/` 子目录，
嵌入变量声明放在组件的 `lua.go`。`scripts/` 目录里不放 `.go` 文件（embed 的同包约束）。
一个组件有多个脚本时，在 `lua.go` 里写一张**归属表**说明每个脚本属于谁、被谁调用——不然
`component/` 会慢慢退化成第二个 `utils/`。

## 跨服务契约只有一个来源

判据一句话：**两个服务读到不同的值会静默出错吗**。会，它就不是某个服务的实现细节，而是契约，只能有一处定义。

| 契约类型 | 放哪 |
|---|---|
| 跨服务 Redis Key | `pkg/rediskey` |
| 跨服务事件 | `pkg/event/*` |
| 单服务 Redis Key | `internal/common/consts/redis` |

配套的两条禁令：

- **Key 与它的 Builder 必须同包**，改了 Prefix 忘改 Builder 会静默错。
- **禁止「万能 Prefix 拼接函数」**（`BuildKey(prefix, a, b string)` 这种）。判据是
  **函数名里能否说出业务对象**：说不出（叫 `BuildKey`）就不是 Key Builder，是把拼字符串的权力
  下放给了每个调用方，等于没有契约。所以 `pkg/rediskey` 里只有
  `FeedBigVGlobal` / `HotFeedDirty(shard)` 这种名字。

判据**不成立**的就别上提。`count:value`、`count:user:profile` 看着像通用缓存，但 Go 代码里只有计数域在用
（内容域拿计数走 `counterservice` RPC），提进 `pkg` 只会让公共库分不清哪些是真契约、哪些是某家的家务事。

## 包名与目录

- 不新增 `utils` `helper` `misc` `base` `shared` 兜底包（**目录名也算**）。这类名字的共同问题是
  **不表达业务对象**，于是什么都能往里放，最后谁都说不清里面有什么、能不能改。能力放进
  `internal/common/component/<能力>/`，名字必须能说出是什么能力。
- `internal/common` 根目录不放 Go 文件，只放有明确归属的子目录（`component` / `consts` / `convert` / `enums`）。
- 只有一层的能力不必建 Component（Component 是给「多入口点共享」用的）；单服务常量也不必上提 `pkg`。

## 生成边界

生成物不可手改——改了下次生成就被覆盖。只改源定义再重新生成：`*.gen.go`（`internal/entity/`）、
`routes.go`（两个 BFF 的 `internal/handler/`）、`*.pb.go` / `*_grpc.pb.go` / RPC client、Swagger、
admin docmeta。生成命令见 [new-interface.md](new-interface.md)。

## 当前的机检覆盖范围

- **全仓规则**（各域都已达标，不分域）：跨服务 internal import、Repository 不引 Logic 与 Component、
  Logic 不引 Repository 实现子包、`pkg` 不反引 `app`、跨服务 Redis Key 不重复定义、
  `internal/common` 根目录无 Go 文件、`common/component` 下无兜底子目录
- **按域规则**（`STRUCT_SCOPES` 变量圈定，**目前只有 `app/rpc/content`**）：cron 不引 logic 与 consumer、
  consumer 不引业务 Repository、Logic 不存完整 `ServiceContext`、无 `utils` 目录、枚举一致性测试存在

其余 8 个服务还没迁移到这些约束（`cron` 还引 `logic`、`consumer` 还引业务 Repository、多处 `utils` 目录）。
某个域清理干净后，把它的根路径加进 `check_structure.sh` 的 `STRUCT_SCOPES` 就放开了——
**别在没清理干净之前把域加进去**，那只会让人习惯性忽略这个检查。
