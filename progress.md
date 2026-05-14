# 会话进度日志

## 当前状态

**最后更新：** 2026-05-14  
**会话 ID：** session-004  
**当前功能：** fix-005 热榜同分翻页修复（已完成）

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
- [x] **fix-003**：gRPC 拦截器日志逻辑反转（B-03）— ServerGrpcInterceptor 改为正向分支，业务错误 Info、系统错误 Error，打印真实 err
- [x] **fix-007**：评论 RPC nil 保护（B-12）— GetUser 后加 `resp != nil && resp.UserInfo != nil` 双重检查，缺失时字段保持空串
- [x] **fix-005**：热榜同分翻页丢失（B-05）— `query_hot_feed_zset.lua` 上界改为包含性，成员级过滤生效

### 进行中

- 无

### 下一步

**剩余 P1 / 杂项：**

- 修复 `fix-002`：JWT Issuer 错误项目名（B-02）—— 一行修改
- 修复 `fix-006`：关注服务不验证被关注用户是否存在（B-11）—— 调 user-rpc 验存在
- 修复 `fix-004`：HTTP 状态码全 200（B-04）—— 多文件
- 修复 `fix-009`：杂项低优先级 Bug（B-06~B-10）—— 魔法数字 / 拼写 / 文件名
- 新增 `sec-002`：Nginx HTTPS + 安全响应头 + limit_req
- 新增 `feat-013`：测试基础设施

---

## 阻塞 / 风险

- [ ] **`pkg/jwt` 无调用方**：JWT 包目前是死代码；实际登录用 Session（Redis Lua）。fix-001 仅恢复包功能。
- [ ] **`fix-002` 未跟进**：Issuer 仍为 "gomall"，留待下次 JWT 相关任务一起处理。
- [ ] **评论缓存可能写入空 userName/userAvatar**：UserRpc 失败时缓存仍写入但 user 字段为空。下次读取需有"空则回源补齐"路径；当前读路径行为待确认。
- [ ] **热榜同分过滤的 memberId 数字 vs 字典序**：当前过滤用 `tonumber(member) >= cursorId` 数字比较，但 Redis 同分内 ZREVRANGEBYSCORE 排序是字典序降序。content_id 若长度不一（如 "5" vs "100"），可能不一致。生产用 snowflake/自增长一致长度时 OK，不属本次修复范围。
- [ ] **OSS 未配置**：`deploy/.env` 中 OSS 相关字段为空，视频封面/头像上传功能不可用
- [ ] **视频转码为占位**：`feat-004` 的 `transcode_status` 始终为 10，HLS 播放不可用
- [ ] **无测试覆盖**：项目零测试，任何修复无法自动化验证

---

## 已做决策（本次新增）

- **fix-005 只动上界，不动过滤器**：description 明确"包含性上界 + 严格 memberId < cursorId 过滤"。现有过滤器 `memberId >= cursorId` 为跳过条件，等价于"只保留 memberId < cursorId"，逻辑已正确，仅因上界排除变成死代码。最小变更面。
- **不修复 memberId 数字 vs lex 比较的潜在不一致**：超出 fix-005 范围，记入风险栏，等内容生成器策略变更（出现变长 ID）时再处理。

---

## 本次会话修改的文件

- `pkg/jwt/token.go` — `SignedString` 与 `ParseWithClaims` 的 keyFunc 改为 `[]byte(secret)`（fix-001）
- `pkg/interceptor/interceptor.go` — `ServerGrpcInterceptor` 错误日志改为正向分支（fix-003）
- `app/rpc/interaction/internal/logic/commentservice/comment_logic.go` — GetUser 返回值 nil 保护（fix-007）
- `app/rpc/content/internal/common/utils/lua/query_hot_feed_zset.lua` — ZREVRANGEBYSCORE 上界改包含性（fix-005）
- `feature_list.json` — fix-001/003/005/007 → done；`last_updated` 与 `_meta_note` 更新
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
2. P0 全部清零，可推进 P1：fix-004 / fix-006 / fix-002 / fix-009
3. fix-007 的"评论缓存空 userName"读路径补齐策略待确认
4. fix-005 的 memberId 数字 vs lex 比较问题已记录，暂未修