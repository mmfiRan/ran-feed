# AGENTS.md

> 本文件帮助使用 AI 在本仓库工作。
> 若本地存在 `.harness/` 目录，先阅读其中的 md 文档并以内容为准；不存在则按本文件内容工作。

**ran-feed** —— 内容/信息流后台系统，Go 微服务（go-zero · gRPC · MySQL · Redis · Kafka）。

---

## 1. 如何用 AI 阅读与开发本项目

1. `pwd` — 确认在项目根目录
2. 完整阅读本文件
3. 准备与验证环境：
   - `go mod download` — 首次克隆后补齐依赖
   - `go build ./...` — 编译所有服务
   - `go vet ./...` — 静态分析
   - `go test ./...` — 运行测试
4. 阅读 [`docs/architecture.md`](docs/architecture.md) — 服务拓扑与代码模式
5. `git log --oneline -8` — 回顾最近提交，了解上下文

---

## 2. 代码约束（项目不变式，违反会引入 Bug）

- **`*.gen.go` 不可手动修改**：`internal/entity/model/*.gen.go`、`internal/entity/query/*.gen.go` 均为 GORM Gen 生成产物
- **`routes.go` 不可手动修改**：`app/front/internal/handler/routes.go` 由 goctl 生成，只改 `.api` 文件后重新生成
- **不在代码中硬编码凭据**：本地开发在根目录放 `.env`（参考部署仓库 ran-feed-docker 的 `.env.example`）
- **软删除必须过滤**：所有 Repository 查询必须加 `IsDeleted.Eq(0)`
- **事务内不调用外部 RPC 或 Redis**：副作用操作放事务提交后执行
- **Redis 优先单命令**：能用 go-zero 提供的单条 redis 命令完成就不写 Lua；确需多步原子操作再用 Lua 脚本（参考 `internal/common/utils/lua/`）

---

## 3. 编码辅助 skill

仓库自带以下 skill，编码 / 测试 / 提交前按需调用（用法与描述由 skill 自身提供）：

- `go-coding` — Go 编码规范与参考（设计模式、缓存、配置等）
- `go-testing` — 测试规范
- `git-commit` — 提交信息规范

---

## 4. 完成标准

一个功能或修复仅在满足全部条件时才视为完成：

- [ ] 目标行为已实现
- [ ] `go build` + `go vet` + `go test` 全部通过
- [ ] 改动范围最小，不做顺手重构
- [ ] 仓库可重启（下一会话可直接继续开发）

---

## 5. 文档地图

| 文件 | 内容 |
|------|------|
| [`docs/architecture.md`](docs/architecture.md) | 服务拓扑、架构、代码模式 |
| [`README.md`](README.md) | 项目简介与部署 |