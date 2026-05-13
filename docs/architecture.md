# 技术架构

## 服务拓扑

```
Client → Nginx(:80) → front-api(:5000)
                         ├── user-rpc(:5003)        — 认证、Session、个人主页、头像
                         ├── content-rpc(:5001)     — 发布、Feed、热榜、OSS 上传凭证
                         ├── interaction-rpc(:5002) — 点赞、评论、关注、收藏
                         └── count-rpc(:5004)       — 聚合计数（点赞/评论/收藏数）

Canal（MySQL CDC）→ Kafka → count-rpc（CanalCountConsumer）
interaction-rpc（点赞事件）→ Kafka → interaction-rpc（LikeConsumer → 计数 delta）
content-rpc XXL-JOB → 热榜 zset 重建（快速：每分钟 / 冷更新：周期性）
```

服务注册/发现：**Etcd**；所有 RPC 客户端配置 `NonBlock: true`。

## 认证流程

`front-api` 使用两个中间件：
- `UserLoginStatusAuthMiddleware` — 要求有效 Session（受保护路由）
- `OptionalLoginMiddleware` — 有 Session 则注入 userId，否则匿名放行

Session Token 是 UUID 字符串，以 Redis Hash 存储，通过 `verify_and_renew_session.lua` 原子验证并续期。**当前请求路径中不使用 JWT 验证**——`pkg/jwt/token.go` 存在已知类型 Bug（见 REVIEW.md §B-01）。

## Feed 架构

三种 Feed，均在 `content-rpc/internal/logic/feedservice/`：

- **推荐流**：全站热榜 Redis zset，XXL-JOB 定时重建；`query_hot_feed_zset.lua` 读全站或快照 zset
- **关注流**：每用户 Redis zset（≤1000 关注者推送写入；大 V 触发拉模式回填）；`query_follow_inbox_zset.lua` + 分布式锁重建
- **用户发布列表**：每用户 Redis zset，首次 miss 懒加载；`query_user_publish_zset.lua`

热度分值由 `pkg/hotrank/formula.go` 时间衰减公式计算；`merge_hot_inc.lua` 做原子浮点精度安全的分值合并。

## 计数架构

`count-rpc` 是 CDC 驱动的计数服务。Canal 监听 `ran_feed_like` / `ran_feed_comment` / `ran_feed_favorite` 表 binlog，发布到 Kafka。`CanalCountConsumer` 将 INSERT/UPDATE/DELETE 转为带符号 delta，通过策略模式（`internal/mq/consumer/strategy/`）应用到 `ran_feed_count_value`。消息去重用 `ran_feed_mq_consume_dedup`。

点赞路径有并行链路：`interaction-rpc` 写 DB 后同时由 `LikeProducer` 发 `LikeEvent` 到 Kafka，`LikeConsumer` 处理更新计数。

## OSS 文件上传

上传使用 **STS 临时凭证**（非直接上传）：
1. 客户端 `POST /v1/content/upload-credentials` → content-rpc 调用阿里云 STS `AssumeRole`
2. 客户端凭证直传 OSS
3. 客户端提交 OSS URL 时发布内容

OSS 策略通过 `internal/common/oss/factory.go` 可插拔，当前仅实现 `AliyunStrategy`。

## 关键代码模式

- **Logic 层**：`internal/logic/<service>/`，每个 RPC 方法一个文件，内嵌 `logx.Logger`，持有 `svcCtx *svc.ServiceContext`
- **Repository 层**：`internal/repositories/` 封装 GORM Gen 查询；软删除必须加 `IsDeleted.Eq(0)` 过滤
- **错误处理**：`pkg/errorx.BizError`（code+message）→ gRPC 服务端拦截器转为 status detail → `pkg/grpcx/converter.go` 在客户端还原
- **Redis Lua 脚本**：Feed 翻页、Session 续期、热度合并等原子操作，通过 `go:embed` 加载于 `internal/common/utils/lua/redis_lua.go`
- **配置注入**：`*.yaml` 使用 `${ENV_VAR}` 占位符，go-zero 启动时从进程环境变量解析