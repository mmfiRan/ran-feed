# 结构规范

跨用例的目录、依赖方向、状态所有权与契约归属。**新增目录、新增组件、新增跨服务常量、拆文件之前读本文。**

机检：`bash .claude/skills/go-coding/scripts/check_structure.sh`，退出码 0 通过、1 有违规并逐条打印文件行号。
`.harness/init.sh` 的第 5 步就是调它。写完自查比等提交时被拦便宜。

本文补的是规则背后的**理由**与容易搞错的地方；可读版速览见
[`docs/architecture.md`](../../../../docs/architecture.md) 的「结构规范」章。

## 两个问题

所有结构规则只回答两件事：

```
1. 这段代码放哪？   →  改需求时要打开几个文件
2. 这份状态归谁管？ →  改规则时会不会漏改
```

要不要收口，只看一句：**改漏了会不会编译失败。** 会（签名、类型），散着放也行，编译器替你盯；
不会（Lua 的 ARGV 顺序、Redis key 拼法、枚举数字、分片数），必须收到一个地方。

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

### 没有构造函数就不是组件

准入四条全中：**有依赖**（去掉 ctx 编译不过）、**有构造**（需要 `New(...)` 且 `svc` 里有装配）、
**无请求状态**、**参与事务不拥有事务**。

最容易判错的是第二条：只有包级纯函数、没有 `New`、没有 struct 的包**不是组件**，
它是 `consts` 加 `convert` 穿了组件的壳，会慢慢变成第二个 `utils/`。机检有一条查这个。

反过来，**有依赖的能力就是组件，哪怕现在只有一个调用方**——它迟早长出第二个调用方
（通常是 Consumer 或 Cron），那时待在 `logic/` 里就取不出来。

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

## 状态所有权

> **一份状态（一个 Redis key / 一张表 / 一个索引）只能由一个 Component 读写。
> Logic、Consumer、Cron 里不出现 redis 原语。**

两条补充：

- 几个 key 共享同一个不变量时归同一个 owner（判据是不变量，不是 key 个数）
- 编解码跟着脚本走，不同脚本各自一份——改 A 脚本的 ARGV 不该被迫动 B 的

为什么单列一条：位置合规不等于所有权清晰。一份状态被 4 处读写，每一处都可能符合上面的分层与
入口点规则，但「这份状态的规则是什么」没有唯一答案，改一条规则要翻 4 个文件，**漏一个不报错**。

没有 owner 的四个症状：

1. 同一个 key builder 在 ≥2 个包里被调用
2. 同一套 Lua ARGV 的拼装有 ≥2 份
3. 出现「脚本抽屉」包——名字说不出业务对象，里面是几个互不相关的脚本
4. 「读缓存 → 抢锁回源 → 写空哨兵 → 续期」的骨架被抄了 ≥2 遍

第 3 条是必然产物不是疏忽：脚本被 N 个包共用，就只能放进一个谁都能引的抽屉里。抽屉是结果不是原因。

落地形状：key builder、Lua、参数拼装、哨兵、TTL、抢锁重建全在包内，
对外方法用业务语义（`Page` `Add` `Remove` `Invalidate`），不用存储语义（`ZAdd` `Eval`）。

**验收：边界切对了，那个共享的脚本抽屉能整包删掉。** 删不掉说明还有状态没有主人。

这条与「跨服务契约只有一个来源」是两个问题：key 定义统一了，仍可能有 4 个包各自调用这个
builder 去读写。

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

脚本分三层，每层由一个变量圈定范围：

- **全仓规则**（各域都已达标，不分域）：跨服务 internal import、Repository 不引 Logic 与 Component、
  Logic 不引 Repository 实现子包、`pkg` 不反引 `app`、跨服务 Redis Key 不重复定义、
  `internal/common` 根目录无 Go 文件、`common/component` 下无兜底子目录
- **按域规则**（`STRUCT_SCOPES`，**目前只有 `app/rpc/content`**）：cron 不引 logic 与 consumer、
  consumer 不引业务 Repository、Logic 不存完整 `ServiceContext`、无 `utils` 目录、枚举一致性测试存在
- **所有权规则**（`OWNERSHIP_SCOPES`，目前 `app/rpc/content`）：Logic 与 Consumer 不操作 Redis、
  `component` 子包必须有构造函数、Consumer 与 Cron 不存 `ctx` 或 `Logger` 字段

其余 8 个服务还没迁到按域与所有权规则（`cron` 还引 `logic`、`consumer` 还引业务 Repository、
多处 `utils` 目录、Redis key 散在 Logic 里）。

某个域清理干净后，把根路径加进对应变量就放开了。三条纪律：

1. **别在没清理干净之前把域加进去**，那只会让人习惯性忽略这个检查
2. 加新规则前先确认它在现有代码上是绿的，加完**故意造一条违规验证它真能变红**。只能绿的检查没有价值
3. **能机检的沉进脚本，脚本是唯一事实源**；文档只写脚本查不了的判据与理由

> 一个已知的边界：`cron` 不在「不许操作 Redis」的扫描范围内——任务级分布式锁属于 Job，
> 是规范允许的（业务缓存重建锁才属于 owner 组件）。这条只能人工核。

## 迁一个服务到上面这些约束

### 纪律

- 一个批次 = 一次提交，批次内不掺第二种操作
- **移动/重命名 与 签名改写 分开提交**：前者 diff 是 rename（只核名字），后者要读逻辑，混了无法审阅
- 语义修正单独提交，用 `fix:` 前缀
- 一个服务做完再做下一个，不并行
- **复制阶段不该需要新的判断**。需要了说明本文有漏，回头补本文再继续，不要现场发明做法

### 批次骨架

| 组 | 动作 | 验收 |
|---|---|---|
| W1 移动 | `common/utils/` 解散 · cron 目录改紧凑小写 · Logic 辅助文件按闭集改名 | `grep -rn '<旧路径>' \| wc -l` = 0 |
| W2 Repository | `ctx` 改首参 · 删 Logger 与 `db` 死字段 · 接口实现分离 + 断言 · `svc` 注入接口 · 补软删除过滤 | `grep -c 'query.SetDefault' internal/svc/*.go` = 1 |
| W3 DO/枚举/常量 | `do` 按准入收敛 · 补 `enum_consistency_test.go` · 常量按四级归位 · 跨服务 key 迁 `pkg/rediskey` | `go test ./internal/common/enums/...` |
| W4 Component | ≥2 入口点需要的实现下沉 · `New` 显式注入 · 按角色切文件 · `.lua` 与 embed 同包 | `find <组件>/scripts -name '*.go' \| wc -l` = 0 |
| W5 所有权 | 每份状态收一个 owner · Logic 与 Consumer 去掉 redis | **脚本抽屉能整包删掉** |
| W6 Logic | 去 `svcCtx` 字段 · 长文件按五条信号拆 | `grep -rn 'svcCtx \*svc.ServiceContext' internal/logic --include='*_logic.go' \| grep -v 'func New' \| wc -l` = 0 |
| W7 MQ/Cron | 删 `svcCtx` `ctx` `Logger` 字段 · 业务下沉 Component · 幂等改原子领取 · 时间窗入口固定一次 | `grep -rn 'internal/mq/consumer\|internal/logic' internal/cron \| wc -l` = 0 |
| W8 收尾 | 删空目录 · 该域加进 `check_structure.sh` 对应 scope | `./.harness/init.sh` 全绿 |

依赖：W1 → 全部；W2 → W3/W6；W4 → W5/W7；W5 → W7。W3 与 W4 可并行。

每批验收基线：`gofmt -l . && go build ./... && go vet ./... && go test ./...`。
`init.sh` 是环境健康门，会话开始与提交前各跑一次，不是每批验收项。

### 编译器证明不了的，必须人工过

| 类 | 怎么验 |
|---|---|
| 幂等 / 重复消费 | 同一 eventID 连发两次，确认第二次不执行业务 |
| 查询结果集变化 | 关键查询跑真库，对比前后行数 |
| 能力等价抽取 | 同一输入跑新旧两条路径对比输出 |
| 并发正确性 | 人工核代码，按加锁判据表逐条对 |

### 路径映射

| 当前 | 目标 |
|---|---|
| `common/utils/<能力>/` | `common/component/<能力>/` |
| `common/utils/lua/` | 拆入各 owner 组件的 `scripts/` |
| `common/utils/<纯函数>.go` | `common/convert/<主题>.go` |
| `logic/<能力>/` · `internal/<能力>/` | `common/component/<能力>/` |
| `internal/types/`（RPC 手写） | `internal/do/` |
| `internal/event/` | `internal/mq/event/` 或 `pkg/event/` |
| `cron/hot_update/` | `cron/hotupdate/` |
| `logic/*_helper.go` | `_mapper.go` / `_source.go` / `cursor.go` / Component |
| `do.XxxDO` · `do.XxxDetail` | `do.Xxx` · `do.XxxResult` |
| `Cron → Consumer.Replay` | 两者共用同一 Component |
| `pkg/*gen`（构建期工具） | `script/generate/` |

### 进度

| 服务 | 状态 |
|---|---|
| `app/rpc/content` | W1–W8 全部完成（试点），三层机检均已打开 |
| 其余 8 个服务 | 未开始，按本文骨架裁剪 |

`content` 的状态归属可作样板：

| 状态 | owner |
|---|---|
| `content:detail` | `contentcache` |
| `feed:user:publish` | `publishbox` |
| `feed:follow:inbox` + `:pull` | `followfeed` |
| `feed:user:favorite` | `favoritebox` |
| `feed:hot:*` | `hotfeed` |
| `feed:bigv:global` | `bigv` |

`feedpub` 与 `feedprojector` 不拥有状态，它们是编排组件，把写委托给上面各 owner。

### 已知陷阱（踩过的，别再踩）

- go-zero 的 `json:",default=..."` 只在 `conf.MustLoad` 读 yaml 时生效。写在 XXL-JOB
  任务参数（走 `encoding/json`）上是死标签，删掉守卫会退化成零值
- `kq.NewPusher` 未传 `kq.WithSyncPush()` 时走 `ChunkExecutor`，`Add` 永远返回 nil，
  写失败只在 go-queue 内部打日志——这种情况下 Producer「返回最终错误」做不到
- handler 全是 Redis / RPC 副作用时不要塞进 `pipeline.RunInTx`，否则违反「事务内不调 RPC/Redis」，
  且幂等顺序会从「先占坑后执行」翻转成「先执行后标记」
