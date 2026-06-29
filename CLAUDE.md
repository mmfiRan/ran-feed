# CLAUDE.md

> 本文件是 Claude Code 在本仓库工作的**路由入口**：只放启动路径、强制规则、项目不变式与文档地图。
> 可复用的编码/提交方法已抽成 skill，状态真相在 `feature_list.json`。**每次新会话开始必须完整阅读。**

**ran-feed** —— 内容/信息流后台系统，Go 微服务（go-zero · gRPC · MySQL · Redis · Kafka）。

---

## 1. 启动工作流（按序执行）

1. `pwd` — 确认在项目根目录
2. 完整阅读本文件
3. `./init.sh` — 编译 + 静态分析 + 测试，失败先修复
4. 阅读 `feature_list.json` — 功能状态唯一真相来源
5. 阅读 `progress.md` — 最近状态与下一步
6. 阅读 `session-handoff.md` — 上一会话交接（如有）
7. `git log --oneline -8` — 回顾最近提交

---

## 2. 核心工作规则（强制）

| # | 规则 |
|---|------|
| R1 | 一次只做一个功能：从 `feature_list.json` 选一条 `not-started` / `in-progress` |
| R2 | 验证后才算完成：未通过 `./init.sh` 不得声称完成，也不得提交 |
| R3 | 不扩大范围：只改与当前条目相关的文件，不做顺手重构 |
| R4 | 会话结束更新 `progress.md`、`feature_list.json`（含 evidence）、必要时 `session-handoff.md` |
| R5 | 保持可重启：收尾时仓库须能让下一会话直接跑通 `./init.sh` |
| R6 | 编码 测试 提交使用对应 project skill（`go-coding` / `go-testing` / `git-commit`），描述由 skill 自身提供 |

---

## 3. 关键约束（项目不变式，违反会引入 Bug）

- **`*.gen.go` 不可手动修改**：`internal/entity/model/*.gen.go`、`internal/entity/query/*.gen.go` 均为 GORM Gen 生成产物
- **`routes.go` 不可手动修改**：`app/front/internal/handler/routes.go` 由 goctl 生成，只改 `.api` 文件后重新生成
- **不在代码中硬编码凭据**：本地开发在根目录放 `.env`（参考部署仓库 ran-feed-docker 的 `.env.example`）
- **软删除必须过滤**：所有 Repository 查询必须加 `IsDeleted.Eq(0)`
- **事务内不调用外部 RPC 或 Redis**：副作用操作放事务提交后执行
- **Redis 优先单命令**：能用 go-zero 提供的单条 redis 命令完成就不写 Lua；确需多步原子操作再用 Lua 脚本（参考 `internal/common/utils/lua/`）

---

## 4. 完成标准 / 升级处理

**完成标准（Definition of Done）** — 一个条目仅在满足全部条件时才能标 `done`：

- [ ] 目标行为已实现
- [ ] `./init.sh` 验证通过（`go build` + `go vet` + `go test`）
- [ ] 完成证据写入 `feature_list.json` 的 `evidence` 字段
- [ ] 仓库可重启

**升级处理** — 遇到以下情况暂停并询问用户：

- 架构决策（新增服务、改 proto、变更 Schema）
- 安全相关（凭据、认证逻辑）
- 依赖变更（修改 `go.mod`）
- 需求不明确 / 同一验证反复失败

---

## 5. 文档地图

| 文件 | 内容 |
|------|------|
| [`docs/architecture.md`](docs/architecture.md) | 服务拓扑、架构、代码模式 |
| [`feature_list.json`](feature_list.json) | 功能状态跟踪（唯一真相来源） |
| [`progress.md`](progress.md) | 会话连续性日志 |
| [`session-handoff.md`](session-handoff.md) | 多会话交接件 |
| [`REVIEW.md`](REVIEW.md) | 代码审查清单 |
| [`init.sh`](init.sh) | 标准启动与验证脚本 |
