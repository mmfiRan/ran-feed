# 会话进度日志

## 当前状态

**最后更新：** 2026-05-14  
**会话 ID：** session-004  
**当前功能：** fix-009 杂项 Bug 打包修复（已完成）

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
- [x] **fix-006**：关注前验证用户存在（B-11）— follow 加 UserRpc.GetUser 校验；unfollow 保持清理语义不验
- [x] **fix-009**：杂项 Bug 打包（B-06~B-10）— TranscodeStatus 常量化 / Kafka 日志格式 / commentStatusDeleted 统一到 consts / Custome→Custom 拼写 / SQL 文件名

### 进行中

- 无

### 下一步

**剩余 P1 / 杂项：**

- 修复 `fix-002`：JWT Issuer 错误项目名（B-02）—— 一行修改
- 修复 `fix-004`：HTTP 状态码全 200（B-04）—— 多文件，按错误类型映射 401/422/500
- 新增 `sec-002`：Nginx HTTPS + 安全响应头 + limit_req
- 新增 `feat-013`：测试基础设施

---

## 阻塞 / 风险

- [ ] **`pkg/jwt` 无调用方**：JWT 包目前是死代码；实际登录用 Session（Redis Lua）。fix-001 仅恢复包功能。
- [ ] **`fix-002` 未跟进**：Issuer 仍为 "gomall"，留待下次 JWT 相关任务一起处理。
- [ ] **评论缓存可能写入空 userName/userAvatar**：UserRpc 失败时缓存仍写入但 user 字段为空。下次读取需有"空则回源补齐"路径；当前读路径行为待确认。
- [ ] **热榜同分过滤的 memberId 数字 vs 字典序**：当前过滤用 `tonumber(member) >= cursorId` 数字比较，但 Redis 同分内 ZREVRANGEBYSCORE 排序是字典序降序。content_id 长度不一时可能不一致。
- [ ] **fix-006 关注路径增加了 user-rpc 同步依赖**：user-rpc 故障时关注接口连带不可用。
- [ ] **OSS 未配置**：`deploy/.env` 中 OSS 相关字段为空
- [ ] **视频转码为占位**：`feat-004` 的 `transcode_status` 始终为 TranscodeStatusPending(10)，HLS 播放不可用
- [ ] **无测试覆盖**：项目零测试

---

## 已做决策（本次新增）

- **fix-009 五个子项一次性打包**：feature 描述本身就是"合并修复"，符合"一次提交对应一个 feature_list 条目"。每个子项体量都很小且互不耦合。
- **commentStatusNormal 不一起迁**：只有 `commentStatusDeleted` 跨包使用（8 个文件），`commentStatusNormal` 只在 comment_logic.go 自己用。仅迁需要跨包共享的，最小变更面（rules.md "不扩大范围"）。
- **TranscodeStatusPending 命名**：CLAUDE.md / feat-004 描述里写"占位为 10（未开始）"，故命名 Pending（待开始/排队）。当转码任务接入后可继续扩展 Running / Done / Failed。
- **B-09 顺带改 Name() 返回值**：`Name()` 返回 "CustomePlugin" 字符串，是 GORM Plugin 注册标识。同步改为 "CustomPlugin"，避免外部代码若已按错误拼写访问会断（当前仓库内无依赖此字符串值）。
- **git mv 重命名 SQL 文件**：保留历史关联，对比 add+delete 更利于追踪。

---

## 本次会话修改的文件

- `pkg/jwt/token.go` — `SignedString` 与 keyFunc 改 `[]byte(secret)`（fix-001）
- `pkg/interceptor/interceptor.go` — 错误日志正向分支（fix-003）
- `app/rpc/interaction/internal/logic/commentservice/comment_logic.go` — GetUser nil 保护（fix-007）
- `app/rpc/content/internal/common/utils/lua/query_hot_feed_zset.lua` — 上界改包含性（fix-005）
- `app/rpc/interaction/internal/logic/followservice/follow_user_logic.go` — UserRpc.GetUser 校验（fix-006）
- `app/rpc/interaction/internal/logic/followservice/unfollow_user_logic.go` — TODO 改说明性注释（fix-006）
- `app/rpc/content/internal/common/consts/content_consts.go` — 新增 TranscodeStatusPending（fix-009 B-06）
- `app/rpc/content/internal/logic/contentservice/publish_video_logic.go` — 引用 TranscodeStatusPending（fix-009 B-06）
- `app/rpc/interaction/internal/mq/producer/like_producer.go` — 日志格式 `%d/%d 次`（fix-009 B-07）
- `app/rpc/interaction/internal/common/consts/comment_consts.go` — 新建，导出 CommentStatusDeleted（fix-009 B-08）
- `app/rpc/interaction/internal/logic/commentservice/{comment_tombstone_fill,comment_user_fill,query_comment_list_logic,query_reply_list_logic,batch_get_comments_logic,refill_comment_cache_logic}.go` — 引用 consts.CommentStatusDeleted（fix-009 B-08）
- `app/rpc/interaction/internal/repositories/comment_repository.go` — 同上 + 删除本地 const（fix-009 B-08）
- `pkg/orm/plugin.go` / `pkg/orm/orm.go` — Custome → Custom（fix-009 B-09）
- `script/sql/ran-feed/content/ran_feed_vidio.sql` → `ran_feed_video.sql` — git mv（fix-009 B-10）
- `feature_list.json` — fix-001/003/005/006/007/009 → done；`last_updated` 与 `_meta_note` 更新
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
2. P0 全部清零；P1 已修 fix-005/006/009，剩 fix-002（一行）/ fix-004（多文件）
3. fix-007 的"评论缓存空 userName"读路径补齐策略待确认
4. fix-005 的 memberId 数字 vs lex 比较问题已记录
5. fix-006 关注路径引入 user-rpc 同步依赖