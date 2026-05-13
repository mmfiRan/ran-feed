# 常用命令

## 构建与运行

```bash
# 一键启动（Docker Compose）
./script/start.sh        # 构建镜像、加载前端包、启动所有服务
./script/stop.sh         # 停止所有服务

# 单独编译
go build ./...           # 编译全部服务
go vet ./...             # 静态分析
go build ./app/front     # 只编译 front-api
go build ./app/rpc/user  # 只编译 user-rpc

# 本地开发（不用 Docker）
cp deploy/.env .env      # 复制并按需编辑（尤其是 OSS 配置）
go run ./app/front       # 启动 front-api
```

每个服务的 Dockerfile 在 `build/<service>.Dockerfile`，二阶段构建：`golang:1.25-alpine` → `alpine:3.20`。  
本地 `.env` 由 `pkg/envx` 在启动时自动加载；各服务配置在 `app/**/etc/*.yaml`，`${VAR}` 由环境变量解析。

## 代码生成

### GORM Gen（ORM 模型）

数据库可访问时运行，输出到 `internal/entity/query/`：

```bash
cd app/rpc/content     && go run ./gen/generator.go
cd app/rpc/user        && go run ./gen/generator.go
cd app/rpc/interaction && go run ./gen/generator.go
cd app/rpc/count       && go run ./gen/generator.go
```

**绝对不要手动修改 `*.gen.go` 文件**，下次生成会覆盖。

### goctl（go-zero HTTP 路由）

修改 `app/front/doc/*.api` 文件后重新生成（`routes.go` 和 Handler 桩由 goctl 管理，不可手编）：

```bash
goctl api go -api app/front/doc/front.api -dir app/front
```

### protoc（gRPC）

```bash
goctl rpc protoc app/rpc/user/proto/user.proto \
  --go_out=app/rpc/user --go-grpc_out=app/rpc/user --zrpc_out=app/rpc/user --style=go_zero --multiple
```