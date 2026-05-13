# 会话进度日志

## 当前状态

**最后更新：** 2026-05-13  
**会话 ID：** session-003  
**当前功能：** 关注流推拉结合 Feature C（feat-021）—— 读时大 V merge

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

### 进行中

- 无

### 下一步

**推拉结合三件套（feat-019/020/021）全部完成。** 可推进的存量 Bug：

- 修复 `fix-001`：JWT 密钥类型错误（B-01）—— P0，JWT 功能完全失效
- 修复 `fix-003`：gRPC 拦截器日志逻辑反转（B-03）—— P0，错误日志全失效
- 修复 `fix-007`：评论 RPC 返回值无 nil 保护（B-12）—— P0，线上 panic 风险
- 修复 `fix-006`：关注服务不验证被关注用户是否存在（B-11）
- 修复 `fix-004`：HTTP 状态码全 200（B-04）
- 修复 `fix-005`：热榜同分翻页丢失（B-05）

---

## 阻塞 / 风险

- [ ] **大 V 缓存 TTL 滞后**：viewer 的"大 V 列表"缓存 TTL=300s，期间用户新关注的大 V 不会立即出现在 merge 中。可接受（5min 容忍）；如需即时一致可在 follow/unfollow 时主动 DEL 该 key（feature 可后续加）
- [ ] **OSS 未配置**：`deploy/.env` 中 OSS 相关字段为空，视频封面/头像上传功能不可用
- [ ] **视频转码为占位**：`feat-004` 的 `transcode_status` 始终为 10，HLS 播放不可用
- [ ] **无测试覆盖**：项目零测试，任何修复无法自动化验证

---

## 已做决策（本次新增）

- **大 V 列表缓存**：用 Redis SET `feed:follow:bigv:{viewerId}`；空集用 sentinel 成员 `"0"` 标识"已计算且空"，避免反复 rebuild；不加分布式锁（thundering herd 多算一次成本可控，正确性不受影响）
- **大 V 候选扫描**：仅扫描 viewer 最近关注的前 500 个（`BigVFolloweesScanLimit`），并行 `mr.WithWorkers(16)` 调 count-rpc.GetCount 筛选，避免 5000 关注 = 5000 RPC 的最坏情形
- **per-request 拉取并发**：每个大 V 一次 Lua（复用 `QueryUserPublishZSetScript`），`mr.ForEach + WithWorkers(16)` 并发，硬上限 `BigVMergeMaxQuery=100`
- **不动 coldBackfill**：缓存 miss 走 DB 已按 author 列表查全部，已天然覆盖大 V
- **不抽公共 `getFollowerCount`**：避免 feedservice ↔ contentservice 跨包依赖，4 行调用本地复刻
- **Merge 正确性**：contentID 是全局自增主键，inbox 与所有 publish zset 共用同一 score 轴；每源取 pageSize+1 个 → 并集去重后排序，全局 top pageSize 必然在并集内

---

## 本次会话修改的文件

- `app/rpc/content/internal/logic/feedservice/bigv_merge_helper.go` — 新增。封装 `loadViewerBigVList` / `computeViewerBigVList` / `listFolloweesCapped` / `writeBigVCache` / `fetchBigVContentIDs` / `queryBigVPublishIDs` / `mergeContentIDs`（feat-021）
- `app/rpc/content/internal/logic/feedservice/follow_feed_logic.go` — `cacheExists` 分支注入 merge 调用（feat-021）
- `app/rpc/content/internal/config/config.go` — `FollowFanOutConfig` 增加 4 个字段（feat-021）
- `app/rpc/content/etc/content.yaml` — 同步 4 个新字段及中文注释（feat-021）
- `app/rpc/content/internal/common/consts/redis/redis_consts.go` — 新增 `RedisFeedFollowBigVPrefix` / `FollowBigVEmptySentinel` 常量 + `BuildFollowBigVKey` 函数（feat-021）
- `feature_list.json` — feat-021 → done，evidence 填写
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
2. **优先推进 P0/P1 存量 Bug**（fix-001 JWT、fix-003 拦截器、fix-007 评论 nil）—— CLAUDE.md 中明确高优先级
3. 关注流推拉结合三件套已闭环：feat-019（ListFollowers RPC）+ feat-020（发布时小账号 fan-out）+ feat-021（读时大 V merge）。后续优化点：
   - follow/unfollow 时主动 DEL viewer 大 V 缓存，让大 V 列表实时反映
   - 大 V 阈值动态化（按时段或按 viewer 分群）
   - 把 `getFollowerCount` 抽到 `internal/common` 让 feed/content 两侧共用（仅在被其他模块也需要时再做）