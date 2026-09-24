# 技术架构

## 服务拓扑

```
Client → Nginx(:80) → front-api(:5000)
                         ├── user-rpc(:5003)        — 认证、Session、个人主页、头像
                         ├── content-rpc(:5001)     — 发布、Feed、热榜、OSS 上传凭证
                         ├── interaction-rpc(:5002) — 点赞、评论、关注、收藏
                         ├── count-rpc(:5004)       — 聚合计数（点赞/评论/收藏数）
                         └── search-rpc(:5006)      — 内容/用户检索（ES）、搜索历史

Canal（destination=count）→ Kafka → count-rpc（CanalCountConsumer → 计数）
Canal（destination=search）→ Kafka(ran-feed-search-canal) → search-rpc（CanalSearchConsumer → ES）
interaction-rpc（点赞事件）→ Kafka → interaction-rpc（LikeConsumer → 计数 delta）
content-rpc XXL-JOB → 热榜 zset 重建（快速：每分钟 / 冷更新：周期性）
search-rpc XXL-JOB → ES 全量重建（SearchReindexJob，初始灌入 + 周期兜底漂移）
```

服务注册/发现：**Etcd**；所有 RPC 客户端配置 `NonBlock: true`。

## 认证流程

`front-api` 使用两个中间件：
- `UserLoginStatusAuthMiddleware` — 要求有效 Session（受保护路由）
- `OptionalLoginMiddleware` — 有 Session 则注入 userId，否则匿名放行

Session Token 是 UUID 字符串，以 Redis Hash 存储，通过 `verify_and_renew_session.lua` 原子验证并续期。**当前请求路径中不使用 JWT 验证**——`pkg/jwt/token.go` 存在已知类型 Bug（见 REVIEW.md §B-01）。

## Feed 架构

四种 Feed，均在 `content-rpc/internal/logic/feedservice/`，归为三类模型：

- **推荐流**（全局排行榜）：主榜 Redis zset（候选池 5000）+ 快照 zset（对外 2000）。`hot_update` 任务两种模式共用同一套算分：增量模式按脏集合算分，全量模式按时间窗扫库重算，主榜为空时增量自动转全量。算分是幂等时点重算，故两模式无需互斥；`query_hot_feed_zset.lua` 读快照/主榜，Redis 不可用时按 `hot_score` 游标查库兜底。
- **关注流**（多时间线合并）：推拉结合。推侧为收件箱 `feed:follow:inbox:{viewer}`，拉侧为拉模式集 `feed:follow:pull:{viewer}`（关注里的大 V）。两结构由**同一次重建**划分产出、同一 TTL 同生同死，推拉并集恒等于关注列表。发布扇出走异步分批消息（`FanOutBatch`），由独立消费者逐批写收件箱；重建加 `DistLocker` 防击穿，等锁超时降级回源查库。
- **用户发布列表**：每用户 Redis zset（`feed:user:publish:{author}`），只装公开内容全量历史，miss 时抢锁回源重建。
- **收藏列表**：每用户 Redis zset（`feed:user:favorite:{owner}`），重建数据源为 interaction 域 `QueryFavoriteList` RPC。

内容详情统一经二级缓存 `content:detail:{id}` 共享；删除/下架只精确失效 L2，索引类结构靠读时自愈。

热度分值由 `pkg/hotrank` 时间衰减公式计算（半衰期 24h），加权互动项与发布时间的加法项相加。

发布与删除的事件链：业务事务内写发件箱 `ran_feed_content_outbox`，Canal 投递 Kafka，feed 消费者按通知式回源查当前状态并把索引同步过去；幂等键用 outbox `event_id`，漏消费由 `outbox.reconcile` 作业补跑。

## 计数架构

`count-rpc` 是 CDC 驱动的计数服务。Canal 监听 `ran_feed_like` / `ran_feed_comment` / `ran_feed_favorite` 表 binlog，发布到 Kafka。`CanalCountConsumer` 将 INSERT/UPDATE/DELETE 转为带符号 delta，通过策略模式（`internal/mq/consumer/strategy/`）应用到 `ran_feed_count_value`。消息去重用 `ran_feed_mq_consume_dedup`。

点赞路径有并行链路：`interaction-rpc` 写 DB 后同时由 `LikeProducer` 发 `LikeEvent` 到 Kafka，`LikeConsumer` 处理更新计数。

## 搜索架构

`search-rpc` 是**只读派生域**（与 count 同类角色）：不拥有业务数据，订阅 content/user 域变更构建 Elasticsearch 索引，对外提供检索。一个服务内同时承载 gRPC 查询 + Canal 消费 + XXL-JOB 重建，挂同一 `ServiceGroup`。

- **索引**：`ran-feed-content`（标题^3 / 简介^2 / 正文，正文不入 `_source` 控体积）、`ran-feed-user`（昵称^3 / 简介）。中文分词用 IK——索引 `ik_max_word`、查询 `ik_smart`。启动时 `EnsureIndices` 幂等建索引。
- **写入（两条）**：①增量——独立 Canal destination `search` 监听 content/article/video/user binlog → Kafka `ran-feed-search-canal` → `CanalSearchConsumer` 按 `content_id` 回源组装整篇文档，判 `已发布+公开+未删除` 则 upsert 否则从索引删；幂等去重在**写 ES 成功之后**才记 `ran_feed_mq_consume_dedup`，避免回源/写 ES 失败被去重吞掉。②全量——`SearchReindexJob`（XXL-JOB）扫 MySQL bulk 灌入，采用**索引别名切换**：先建新物理索引灌满，再原子把别名切过去并删旧索引，期间旧索引持续可读，且天然不残留孤儿文档。两条均以源行/事件时间作 ES external version 防乱序覆盖（增量路径统一用 canal 事件 ts，单分区内单调）。代码只认 `ran-feed-content` / `ran-feed-user` 两个别名。
- **查询**：`front /v1/search/{content,user}`（`OptionalLoginMiddleware` 注入 viewerId）→ search-rpc 构造 `bool{multi_match + filter}` 查 ES，返回 ranked IDs + 高亮 + score；front **复用 feed 拼装链**富化——内容走 content-rpc `BatchGetContentItems`（内部即 feed 的 `assembleItems`），用户走 user-rpc + follow 关注状态。search-rpc 只管"匹配+排序→ID"，展示与 viewer-state 留在 front。
- **搜索历史**：`ran_feed_search_history`（MySQL）仅登录用户，`UNIQUE(user_id,keyword)` 去重提到最前，搜索时由 front 隐式异步记录。
- **降级**：ES 查询失败 front 返回空 + 错误码不阻断页面；ES 写失败非致命，靠重建 job 补。

## OSS 文件上传

上传使用 **STS 临时凭证**（非直接上传）：
1. 客户端 `POST /v1/content/upload-credentials` → content-rpc 调用阿里云 STS `AssumeRole`
2. 客户端凭证直传 OSS
3. 客户端提交 OSS URL 时发布内容

OSS 策略通过 `internal/common/oss/factory.go` 可插拔，当前仅实现 `AliyunStrategy`。

## 关键代码模式

- **Logic 层**：`internal/logic/<service>/`，每个 RPC 方法一个文件，内嵌 `logx.Logger`，持有 `svcCtx *svc.ServiceContext`
- **Repository 层**：`internal/repositories/` 封装 GORM Gen 查询；软删除过滤统一用枚举（`IsDeleted.Eq(enums.NotDeleted.Int32())`，非 0 判别列用 `Int64()`）
- **错误处理**：`pkg/errorx.BizError`（code+message）→ gRPC 服务端拦截器转为 status detail → `pkg/grpcx/converter.go` 在客户端还原
- **Redis Lua 脚本**：Feed 翻页、Session 续期、热榜快照重建、大 V 成员批量判定、收件箱与发件箱写入等原子操作，通过 `go:embed` 加载于各域 `internal/common/utils/lua/redis_lua.go`
- **配置注入**：`*.yaml` 使用 `${ENV_VAR}` 占位符，go-zero 启动时从进程环境变量解析