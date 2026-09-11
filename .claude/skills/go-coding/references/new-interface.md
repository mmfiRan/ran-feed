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
- **请求字段按是否必传选类型**：
  - 必传参数直接用基本数据类型，配 `validate:"required,..."` 做必填校验
  - 可选参数一律用指针（`*int32` / `*int64` / `*string`），tag 加 `,optional`，配 `validate:"omitempty,..."`
- **可选参数不要用 0 空串当"未传"的哨兵值**：go-zero 的 form 解析对指针字段未传即保持 nil，
  指针能区分"未传"（nil）与"传了 0"，也正好直接映射到下游 RPC 的 optional 字段
- **可选字段的 `validate` 必须带 `omitempty` 前缀**（如 `validate:"omitempty,gt=0"`、`validate:"omitempty,oneof=10 20"`）：
  go-playground/validator 对 nil 指针不会自动跳过校验，漏写 `omitempty` 会导致不传即 400；
  写了才是"未传跳过 传了才校验"，能挡住非法值
- **要留意零值与"未传"业务含义不同的字段 这类字段必须用指针**：例如"没有库存记录"和"库存为 0"
  是两种截然不同的业务处理，用哨兵值会把两者抹平，只有指针能把 nil 传进 logic 分支判断
- 路由按服务分组，handler 名与 logic 一一对应

## 后台 admin-api RBAC 元数据（改 admin 的 .api 必跑）

admin-api 的路由级权限校验不写死在代码里,而是**以 `.api` 每条路由的 `@doc` 为事实源**:

```
@doc (
    description: "角色列表 分页"
    permission: "admin:role:list"   // 需要鉴权的路由声明所需权限点 只登录的(登录/登出/me)不写
)
@handler ListRoles
get /roles (AdminRoleListReq) returns (AdminRoleListRes)
```

`pkg/rbacgen`(独立 module)解析 `.api` 把每条路由 `@doc` 的 key-value 生成到 `app/admin/internal/docmeta`;
`main.go` 全局 `server.Use(docmeta.Inject)` 把当前路由 `@doc` 注入 ctx;`AdminRbacMiddleware` 从 ctx 读
`permission` 校验。**取不到 permission 直接 403(fail-closed),不是降级为只验登录**:漏声明只会让接口
谁都进不去,需在 `.api` 补 `@doc` 后重跑 rbacgen。登录/登出/me 不带 permission 是因为它们没挂
`AdminRbacMiddleware`(见 auth.api 的 middleware 列表),不是因为"未声明"。

> **注意:`.api` 路由不要用路径参数(`/roles/:id`)。** docmeta 的 key 是 `"METHOD /path"` 字面量,
> 而 `docmeta.Inject` 用 `r.URL.Path` 精确匹配,runtime 的 `/roles/123` 匹配不上
> `"DELETE /v1/admin/roles/:id"`,该路由会因取不到 permission 而整体 403。rbacgen 会对此打警告。
> 需要 REST 风格路径时,得先把 `Inject` 改成按 `/` 分段通配匹配。

**强制:凡改动 `app/admin` 下任意 `.api`(增删路由或改 permission),在上面 goctl 两步之外,必须再跑一次
rbacgen 重生成 docmeta**,否则权限映射与路由脱节(改了权限不生效 / 新路由漏鉴权):

```bash
cd pkg/rbacgen && go run . -api ../../app/admin/doc/admin.api -out ../../app/admin/internal/docmeta -pkg docmeta
```

## gRPC（服务间）

改 `app/rpc/<svc>/proto/<svc>.proto` 后重新生成：

```bash
goctl rpc protoc app/rpc/user/proto/user.proto \
  --go_out=. --go-grpc_out=. --zrpc_out=app/rpc/user \
  --go_opt=module=ran-feed --go-grpc_opt=module=ran-feed --module=ran-feed \
  --style=go_zero --multiple --name-from-filename
```

（`user` 替换为 `count` / `interaction` / `search` / `content` / `notification` / `admin`）

命令三要素，缺一都会出错：

- `--go_out=. --go-grpc_out=.` 是模块根（仓库根），不是 `app/rpc/<svc>`；写错会把生成文件放到错误位置
- `--go_opt=module=ran-feed` / `--go-grpc_opt=module=ran-feed` 让 protoc 从 go_package 剥掉 `ran-feed/` 前缀再拼到 `--go_out` 后；漏掉会在仓库根多出一个 `ran-feed/` 目录
- `--name-from-filename` 让服务名取文件名 `<svc>`；否则 `package` 加了 `ranfeed.` 前缀后，服务名会变成 `ranfeedadmin` 这类

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