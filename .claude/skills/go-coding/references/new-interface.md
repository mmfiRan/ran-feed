# 新增接口与代码生成（API / RPC / ORM 必须用 CLI 生成）

本文是本项目接口与模型**代码生成命令的唯一来源**。新增或修改对外接口时，
**先改定义文件再用 goctl 生成，绝不手写或手改生成产物**
（`routes.go` handler 桩 `*.pb.go` `*_grpc.pb.go` client `*.gen.go`）。生成后只在 logic 里写业务。

> 修改 `.proto`（新增/变更 RPC 或 Schema）属于 CLAUDE.md 第 4 节「升级处理」，先与用户确认再动手。

## HTTP API（front 网关）

改 `app/front/doc/<svc>/<svc>.api`（由 `front.api` import 汇总）后，两步都要跑：先生成代码，再重生成 swagger：

```bash
# 1 生成 handler/routes/logic 桩 统一 go_zero 文件名风格
goctl api go --api app/front/doc/front.api --dir app/front --style=go_zero

# 2 改了 api 必须重生成 swagger 文档 保持 front.json 与接口一致
goctl api swagger --api app/front/doc/front.api --dir app/front/swagger --filename front
```

格式约束：
- **生成必须带 `--style=go_zero`**（与 rpc 一致），保持文件名风格统一
- **改了 `.api` 必须同步重生成 swagger**（`app/front/swagger/front.json`），不可只生成代码漏掉文档
- 类型名 PascalCase，请求/响应固定 `XxxReq` / `XxxRes` 后缀
- 字段 Go 侧 PascalCase，JSON tag snake_case（`json:"user_id"`）
- 请求字段用指针 加 `,optional`，并带 `validate:"..."` 标签做参数校验
- 路由按服务分组，handler 名与 logic 一一对应

## gRPC（服务间）

改 `app/rpc/<svc>/proto/<svc>.proto` 后（保持 `--style=go_zero --multiple` 统一文件名风格）：

```bash
goctl rpc protoc app/rpc/user/proto/user.proto \
  --go_out=app/rpc/user --go-grpc_out=app/rpc/user --zrpc_out=app/rpc/user --style=go_zero --multiple
```

格式约束：
- message 与 rpc 方法 PascalCase，请求/响应固定 `XxxReq` / `XxxRes` 后缀
- 字段 snake_case（`target_id` `user_id`）
- 一个 RPC 配一对 `XxxReq` / `XxxRes`，不复用模糊的通用 message

## ORM 模型（GORM Gen）

改了数据库 Schema 后，在数据库可访问时重新生成模型，输出到 `internal/entity/query/`：

```bash
cd app/rpc/content     && go run ./gen/generator.go
cd app/rpc/user        && go run ./gen/generator.go
cd app/rpc/interaction && go run ./gen/generator.go
cd app/rpc/count       && go run ./gen/generator.go
```

**绝对不要手动修改 `*.gen.go`**，下次生成会覆盖。