# 会话进度日志

## 当前状态

**最后更新：** 2026-05-13  
**会话 ID：** session-002  
**当前功能：** 登录优化 + 内容发布 P0 + 关注流 P0 修复

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
- [x] **feat-019**：推拉结合 Feature A — interaction.proto 新增 ListFollowers RPC + repo 方法 ListFollowersByCursor；proto 已重生成；为 Feature B/C 准备好基础设施
- [x] **feat-020**：推拉结合 Feature B — content-rpc 配置新增 FollowFanOut；新增 fanout_helper.go，afterPublish 中 threading.GoSafe 异步查粉丝数判断大 V，小账号分批拉 follower 列表并批量写 inbox；大 V 跳过

### 进行中

- 无

### 下一步

**关注流推拉结合架构改造（已确认方案）：**
- ✅ Feature A：ListFollowers RPC（feat-019）已完成
- ✅ Feature B：发布时 fan-out（feat-020）已完成
- ⬜ Feature C（feat-021）：FollowFeed 读路径识别大 V → 单独查大 V publish zset → 与 inbox 按 score 合并去重分页

**已决策的关键参数：**
- 粉丝数源：复用 count-rpc 的 `FOLLOWED:USER:{userId}`（已由 Canal 自动维护）
- fan-out 机制：goroutine 异步（threading.GoSafe + bg ctx + 30s 超时）
- 大 V 阈值：config.yaml `FollowFanOut.BigVFollowerThreshold`，默认 5000

**其他存量 Bug 待修：**
- 修复 `fix-006`：关注服务不验证被关注用户是否存在（B-11）
- 修复 `fix-007`：评论 RPC 返回值无 nil 保护（B-12，线上 panic 风险）
- 修复 `fix-001`：JWT 密钥类型错误（B-01）
- 修复 `fix-003`：gRPC 拦截器日志逻辑反转（B-03）

---

## 阻塞 / 风险

- [ ] **关注流当前仍是纯拉模式**：发布时不写 follower inbox，新内容直到缓存淘汰才能被看到。短期靠 fix-010 缓解首次体验，根本解决需要推拉架构改造
- [ ] **OSS 未配置**：`deploy/.env` 中 OSS 相关字段为空，视频封面/头像上传功能不可用
- [ ] **视频转码为占位**：`feat-004` 的 `transcode_status` 始终为 10，HLS 播放不可用
- [ ] **无测试覆盖**：项目零测试，任何修复无法自动化验证

---

## 已做决策

- **缓存更新策略**：DB 事务提交后再做 Redis 更新（best-effort），失败只记日志，缓存由懒加载/定时任务重建（rules.md 明文约束："数据库事务内不调用外部 RPC 或 Redis"）
- **关注流架构方向**：长期目标推拉结合（业界主流），短期先修拉模式下的体感 Bug
- **异步任务 ctx 规范**：goroutine 内必须用 `context.WithTimeout(context.Background(), ...)`，并通过 `NewXxxLogic(bgCtx, svcCtx)` 创建新 logic 实例，避免请求 ctx cancel 影响异步工作
- **HTTP 状态码策略**：`fix-004` 修复后，错误响应改为语义化状态码（401/422/500）

---

## 本次会话修改的文件

- `app/rpc/user/internal/logic/userservice/login_logic.go` — 用户状态校验 + 错误信息统一（sec-001）
- `app/front/internal/middleware/userloginstatusauth_middleware.go` — token 提取修复 + parseSessionTTL 死代码清理
- `app/rpc/user/internal/repositories/user_repository.go` — GetByID / BatchGetByIDs 补软删除过滤（fix-008）
- `app/rpc/content/internal/logic/contentservice/publish_article_logic.go` — Redis 移出事务，新增 afterPublish 方法（fix-011）
- `app/rpc/content/internal/logic/contentservice/publish_video_logic.go` — Redis 移出事务，新增 afterPublish 方法（fix-011 对称补完）
- `app/rpc/content/internal/logic/feedservice/follow_feed_logic.go` — 首次访问 DB 兜底 + coldBackfill 接线 + 真异步重建（fix-010）
- `app/rpc/interaction/proto/interaction.proto` — 新增 ListFollowers RPC（feat-019）
- `app/rpc/interaction/interaction/*.pb.go` / `client/*/follow_service.go` / `server/followservice/follow_service_server.go` — goctl 重新生成
- `app/rpc/interaction/internal/repositories/follow_repository.go` — 新增 ListFollowersByCursor 方法（feat-019）
- `app/rpc/interaction/internal/logic/followservice/list_followers_logic.go` — 新增（feat-019）
- `app/rpc/content/internal/config/config.go` — 新增 FollowFanOutConfig（feat-020）
- `app/rpc/content/etc/content.yaml` — 新增 FollowFanOut 配置块（feat-020）
- `app/rpc/content/internal/logic/contentservice/fanout_helper.go` — 新增 fanOutToFollowersAsync 异步推送实现（feat-020）
- `app/rpc/content/internal/logic/contentservice/publish_article_logic.go` — afterPublish 接入 fan-out（feat-020）
- `app/rpc/content/internal/logic/contentservice/publish_video_logic.go` — afterPublish 接入 fan-out（feat-020）
- `feature_list.json` — sec-001 / fix-008 / fix-010 / fix-011 / feat-019 / feat-020 全部 done；feat-021 待推进
- `docs/rules.md` — 设计模式章节从 2 个扩充到 8 个（Strategy/Factory/Repository/Middleware/FunctionalOptions/PubSub/Adapter/TemplateMethod，附触发场景+反例）；新增"Git 提交规范"章节（Conventional Commits + 与 feature_list 联动约定）
- `progress.md` — 本次会话记录

---

## 完成证据

- [ ] 测试通过：`（尚无测试）`
- [x] 编译检查：`go build ./...` 通过
- [x] 静态分析：`go vet ./...` 通过
- [x] `./init.sh` 全流程通过（已完成 14/29）

---

## 下次会话注意事项

1. 跑 `./init.sh` 确认仍干净
2. **推进 Feature C（feat-021）**：FollowFeed 读路径加入大 V merge
   - 取 viewer 的 followee 列表
   - 对每个 followee 查粉丝数，识别大 V
   - 大 V 的 publish zset 单独查最近 N 条
   - 与 inbox 结果按 score 合并、去重、按 cursor 分页
   - cursor 设计要兼容两种数据源
3. 或继续推进 fix-006 / fix-007 / fix-001 / fix-003 等其他存量 Bug