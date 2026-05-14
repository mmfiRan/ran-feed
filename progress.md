# 会话进度日志

## 当前状态

**最后更新：** 2026-05-14  
**会话 ID：** session-004  
**当前功能：** fix-001 JWT 密钥类型修复（已完成）

---

## 状态

### 已完成

- [x] 初始项目发布（feat-001 ~ feat-012 全部上线）
- [x] 全量代码审查报告生成（`REVIEW.md`）
- [x] Harness 工程初始化（`CLAUDE.md` / `feature_list.json` / `progress.md` / `init.sh`）
- [x] **sec-001**：登录接口防用户枚举 — 统一错误信息为"手机号或密码错误"，新增用户状态（禁用账号）校验
- [x] **fix-008**：软删除过滤不一致 — `GetByID` / `BatchGetByIDs` 补加 `IsDeleted.Eq(0)`
- [x] **middleware 优化**：`extractToken` 修复"Bearer"无 token 时误返回问题；`parseSessionTTL` 移除永远失败的死代码分支
- [x] **fix-011**：内容发布事务内 Redis 调用 — `publish_article_logic.go` 与 `publish_video_logic.go` 均完成对称修改，afterPublish 封装 Redis 更新
- [x] **fix-010**：关注流 P0 — 首次访问改为同步 DB 兜底返回首屏；coldBackfill 接入缓存 miss 路径；inbox 重建改为真异步（不阻塞请求）
- [x] **feat-019**：推拉结合 Feature A — interaction.proto 新增 ListFollowers RPC + repo 方法 ListFollowersByCursor
- [x] **feat-020**：推拉结合 Feature B — content-rpc 发布时小账号 fan-out，大 V 跳过
- [x] **feat-021**：推拉结合 Feature C — FollowFeed 读路径识别大 V，并 merge publish zset
- [x] **fix-001**：JWT 密钥类型错误（B-01）— `pkg/jwt/token.go` SignedString 与 ParseWithClaims keyFunc 均改为 `[]byte(secret)`

### 进行中

- 无

### 下一步

**P0 剩余存量 Bug：**

- 修复 `fix-003`：gRPC 拦截器日志逻辑反转（B-03）—— P0，错误日志全失效
- 修复 `fix-007`：评论 RPC 返回值无 nil 保护（B-12）—— P0，线上 panic 风险

**P1：**

- 修复 `fix-002`：JWT Issuer 错误项目名（B-02）—— 顺手在下次 JWT 相关改动时一起处理
- 修复 `fix-006`：关注服务不验证被关注用户是否存在（B-11）
- 修复 `fix-004`：HTTP 状态码全 200（B-04）
- 修复 `fix-005`：热榜同分翻页丢失（B-05）

---

## 阻塞 / 风险

- [ ] **`pkg/jwt` 无调用方**：本次修复仅恢复包功能可用性，但实际登录流程使用的是 Session（Redis Lua），JWT 包目前是死代码。等接入 JWT 鉴权（如 feat-016 通知或后台管理）时才会用到。
- [ ] **`fix-002` 未跟进**：Issuer 仍硬编码为 "gomall"，等下次 JWT 相关任务一起处理。
- [ ] **OSS 未配置**：`deploy/.env` 中 OSS 相关字段为空，视频封面/头像上传功能不可用
- [ ] **视频转码为占位**：`feat-004` 的 `transcode_status` 始终为 10，HLS 播放不可用
- [ ] **无测试覆盖**：项目零测试，任何修复无法自动化验证

---

## 已做决策（本次新增）

- **不一并修 fix-002**：用户明确选 fix-001，按 rules.md "每次只做一个功能 / 一次提交对应一个 feature_list 条目" 原则，Issuer 留到下次。
- **不动 `interface{}` → `any` 的 lint 提示**：那是预先存在的代码风格提示，不在 fix-001 范围内（rules.md "不扩大范围"）。

---

## 本次会话修改的文件

- `pkg/jwt/token.go` — `SignedString` 与 `ParseWithClaims` 的 keyFunc 改为 `[]byte(secret)`（fix-001）
- `feature_list.json` — fix-001 → done，evidence 填写；`last_updated` 与 `_meta_note` 更新
- `progress.md` — 本次会话记录

---

## 完成证据

- [ ] 测试通过：`（尚无测试）`
- [x] 编译检查：`go build ./...` 通过
- [x] 静态分析：`go vet ./...` 通过
- [x] `./init.sh` 全流程通过

---

## 下次会话注意事项

1. 跑 `./init.sh` 确认仍干净
2. **优先推进剩余 P0**：fix-003（gRPC 拦截器日志反转）、fix-007（评论 RPC nil 保护）
3. fix-002（JWT Issuer="gomall"）建议与下一个 JWT 相关任务一起改，避免一行修改单独成 commit