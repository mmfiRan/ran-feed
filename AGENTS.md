# AGENTS.md

> 本地存在 `.harness/` 时，先读 `.harness/DEVELOPER.md`；该文件补充本文，不放宽下面的约束。不存在则跳过。

## 项目结构

- `app/front` — C 端网关（BFF），聚合各 RPC 组装响应
- `app/admin` — 后台管理端BFF，含 RBAC 权限校验与操作审计
- `app/rpc/*` — 各域 RPC 服务（gRPC 通信，Etcd 服务发现）：
  - `user` 认证 / Session / 个人主页；
  - `content` 发布 / Feed / 热榜 / OSS 凭证
  - `interaction` 点赞 / 评论 / 关注 / 收藏；
  - `count` 聚合计数（Canal 驱动）
  - `search` 内容与用户检索（ES）；
  - `notification` 通知（SSE 实时层）；
  - `admin` 后台管理域
- `pkg/*` — 跨服务公共代码（consts / enums / event / rediskey / errorx / grpcx 等）

服务内分层：`Handler → Logic → Repository`，跨用例能力落 `internal/common/component/`，依赖经 `ServiceContext` 注入，依赖只能向下：

- Handler 只做参数绑定与响应序列化，不含业务逻辑
- Logic 负责业务编排，调 Repository 与外部 RPC，不直接写 SQL
- Repository 只做数据读写，返回数据或原始错误，不含业务判断
- Component 只接自己的 Config 子结构，无请求状态，不拥有事务边界

## 开发与验证

```bash
go build ./...
go vet ./...
go test ./...
```

- 提交前必须通过以上命令；跑单个包：`go test ./app/rpc/user/...`
- 本地起服务依赖 MySQL / Redis / Etcd / Kafka，凭据放根目录 `.env`（不入库）
- 未执行的验证要如实说明，不把未验证的改动描述为完成

## 代码边界

**禁止手改生成文件**——重新生成会被覆盖，只改源定义再重新生成：
  - `*.gen.go`（`internal/entity/model/`、`internal/entity/query/`）
  - `routes.go`（`app/front/internal/handler/`、`app/admin/internal/handler/`）
  - protobuf 生成（`*.pb.go`、`*_grpc.pb.go`、RPC client）、Swagger、admin docmeta

**禁止越层与反向依赖：**

- Repository 引 Logic 或 Component；Logic 引 Repository 实现子包（只允许引 `internal/repositories` 接口包）
- Consumer 与 Cron 互相调用或引 `logic/`；Consumer 引业务 Repository
- `pkg/*` 反向引 `app/*`；一个服务 import 另一个服务的 `internal`
- Logic 保存完整 `ServiceContext`；Cron / Consumer 保存 `ctx` 字段
- 新增 `utils` `helper` `misc` `base` `shared` 兜底包（目录名同禁）
- 同一个跨服务 Redis Key 或事件契约在多处定义（应走 `pkg/rediskey` / `pkg/event`）

**先问：**

- 改 `.proto` / 数据库 Schema / 新增服务 / 新增依赖 / 认证安全逻辑

**必须：**

- 有 `is_deleted` 字段的表，Repository 查询过滤未删除（`IsDeleted.Eq(0)`）
- 事务内不调 RPC / Redis，缓存失效、消息发送等副作用放事务提交后
- 跨域数据走对应 RPC 或 Canal 同步链路，不直读别域的表
- 一份状态（Redis key / 表 / 索引）只由一个 Component 读写，Logic 与 Consumer 里不出现 redis 原语
- 结构约束见 [`docs/architecture.md`](docs/architecture.md)「结构规范」；机检入口是 `./.harness/init.sh` 的结构静态检查

## 编码辅助

- `go-coding` / `go-testing` / `git-commit`（用法见各 skill 的 description）

## 文档地图

- [`docs/architecture.md`](docs/architecture.md) — 服务拓扑、架构、代码模式、结构规范
- [`README.md`](README.md) — 项目简介与部署