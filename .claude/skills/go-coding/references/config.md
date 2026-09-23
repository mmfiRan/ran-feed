# 配置新增

新增配置项时：

- 在服务的 `internal/config` Config 结构体加字段，用 `json:",default=..."` 给**安全默认值**
- 在 `app/**/etc/*.yaml` 写入对应值（`${VAR}` 由环境变量解析）
- **零值要能安全关闭该功能**（如 `WindowSeconds<=0` 关登录限流、阈值 0 关 fan-out），不要让漏配导致崩溃
- 凭据类绝不写进 yaml 或代码，走 `.env` 环境变量（见 CLAUDE.md 关键约束）

```go
// 默认值写在 tag 上 漏配也能安全启动
type LoginRateLimitConfig struct {
    WindowSeconds int64 `json:",default=300"`
    MaxAttempts   int64 `json:",default=5"`
}
```