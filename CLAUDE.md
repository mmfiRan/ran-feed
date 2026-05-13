# CLAUDE.md

本文件为 Claude Code（claude.ai/code）在此仓库中工作提供指导，**每次新会话开始时必须完整阅读**。

> ran-feed —— 内容/信息流后台系统，Go 微服务（go-zero · gRPC · MySQL · Redis · Kafka）

---

## 启动工作流

每次新会话按顺序执行，不可跳过：

1. `pwd` — 确认在项目根目录
2. 阅读本文件
3. `./init.sh` — 编译 + 静态分析，失败则先修复
4. 阅读 `feature_list.json` — 确认当前功能状态
5. 读取 `claude-progress.md`，了解最新已验证状态和下一步
6. 阅读 `docs/rules.md` 查看编码规范
7. `git log --oneline -8` — 回顾最近提交

---

## 已知关键 Bug（P0/P1，优先修复）

| ID | 文件 | 问题 |
|----|------|------|
| B-01 | `pkg/jwt/token.go:29` | JWT 传 `string` 而非 `[]byte`，功能完全失效 |
| B-03 | `pkg/interceptor/interceptor.go:54` | 拦截器日志条件取反，错误日志全部失效 |
| B-04 | `pkg/result/result.go` | 所有 HTTP 错误响应均返回 200，监控失效 |
| B-05 | `query_hot_feed_zset.lua` | 热榜同分翻页丢失内容 |
| B-11 | `follow_user_logic.go` | 关注不验证被关注用户是否存在 |
| B-12 | `comment_logic.go:112` | RPC 返回值无 nil 保护，线上 panic 风险 |

完整 Bug/安全/优化列表见 [`REVIEW.md`](REVIEW.md)。

---

## 文档地图

| 文件 | 内容 |
|------|------|
| [`docs/rules.md`](docs/rules.md) | 工作规则、关键约束、完成标准、会话结束清单、升级处理 |
| [`docs/commands.md`](docs/commands.md) | 构建、运行、代码生成命令 |
| [`docs/architecture.md`](docs/architecture.md) | 服务拓扑、认证流程、Feed/计数/OSS 架构、代码模式 |
| [`feature_list.json`](feature_list.json) | 功能状态跟踪（真实来源） |
| [`progress.md`](progress.md) | 会话连续性日志 |
| [`init.sh`](init.sh) | 标准启动与验证脚本 |