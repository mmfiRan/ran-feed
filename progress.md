# 会话进度日志

## 当前状态

**最后更新：** 2026-06-XX  
**会话 ID：** session-008  
**当前功能：** feat-009 热榜算分模型重构 — 按 HOT_FEED_DESIGN 切到方案A（log10 加法时间项 + Set 脏集合 + 回查 count 总量），init.sh 全绿

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
- [x] **fix-002**：JWT Issuer 错误项目名（B-02）— `pkg/jwt/token.go:24` `"gomall"` → `"ran-feed"`
- [x] **fix-004**：HTTP 错误状态码场景化（B-04）— 鉴权失败 401 / 系统未知错误 500；业务错误与校验错误保持 200 由前端按 body code 处理
- [x] **feat-013**：测试基础设施 — 5 个测试文件，30 个 case；testify v1.11.1 + miniredis v2.38.0 作为测试依赖；init.sh 测试步骤从"跳过"改为真正执行
- [x] **sec-001 收口**：登录限频 — ZSET 滑动日志，per-mobile；Check/Record/Clear 三段式；config 零值关闭；2 个新单测
- [x] **fix-012**：热榜 Lua 同分过滤改字典序 — `member >= cursor` 与 Redis 同分组真实排序对齐；删除 cursorId/tonumber(member) 无用变量；新增回归测试覆盖跨位数 content_id 场景；同步发现 latest 分支 false-concat 边界 bug 已记录为后续条目
- [x] **sec-002**：Nginx HTTPS + 安全头 + limit_req — 默认启用 5 个安全响应头 + 分层限流（api 10r/s + login 1r/s）；HTTPS 走 ssl.conf.example 模板（含 80→443 重定向、HSTS、http2 on）默认禁用；证书目录 + .gitignore + README 启用 5 步流程；docker run nginx -t 语法验证通过
- [x] **fix-013**：审计 P0 四点集中修复 — favorite Upsert WithResult 吞错（闭包 createErr 回传）/ like 与 unlike Kafka 发送从 l.ctx 改 bg + 5s timeout / follow 已定义但未接线的 3s timeout 终于用上 / query_favorite_info GetCount resp nil 检查与 like 模块对齐
- [x] **fix-014**：审计 P1 三点集中修复 — unlike 解耦 content-rpc（失败降级 contentUserID=0 继续） / canal 延迟缓存清理换 bg ctx + 5s timeout / comment & reply 列表 nextCursor 用 parseInt64 + hasMore-with-zero-cursor 防御分支
- [x] **fix-015**：审计 P2 三点集中修复 — content.go xxl-job 启动失败 os.Exit(1) 让 supervisor 拉起（ctx.Canceled 视为正常停机） / batch_get_comments fillObjCacheBestEffort 加 per-call 5s timeout 防 goroutine 微泄漏 / query_hot_feed_zset.lua latest 分支识别 Lua false（防 concat 崩溃）+ 新增 fall-through 回归测试
- [x] **feat-009 重构（热榜算分模型切方案A）**：按 HOT_FEED_DESIGN 把乘法指数衰减公式换为加法时间项。(1) pkg/hotrank 新增 AdditiveTime：score=log10(max(加权,1))+发布秒/S，S=半衰期秒/log10(2)，抗霸榜抗刷量、0互动靠时间项进TopN、ZADD时点覆盖自愈漂移（修复旧版「对增量取log再累加」的 Σlog(Δ)≠log(ΣΔ) 数学缺陷）+ 8 个单测；(2) 记账解耦：count-rpc canal 消费者由 HINCRBY 加权delta 改 SADD feed:hot:dirty:{id%64}（删 heatScoreDeltaByBiz/writeHotIncrement→markHotDirty），发布种子改 SAdd；(3) 快更重写：FreezeHotDirtyScript 原子 RENAME 活跃桶→proc桶（双缓冲根治边读边写丢事件）→SSCAN→BatchGetRecommendByIDs过滤删/私有(ZREM)→count BatchGetCount回查总量→算全分ZADD覆盖→裁剪TopN→切快照→DEL proc；(4) 冷更 calcScore 改 AdditiveTime 同口径，清理扩展 dirty+proc+旧inc 桶。读路径/proto/front 端点不变。./init.sh 全绿。
- [x] **B-05 复核（不改）**：计划原拟把 query_hot_feed_zset.lua 同分切分改数值比较，复核 fix-012 测试后确认现状 lex 字典序与 ZREVRANGEBYSCORE 同分组真实排序一致才是正确的；改数值会重现「第二页重复+丢内容」的 fix-012 bug。故保留现状，作废该步。设计 §7「ID 严格数值递减」需 R-05 统一 cursor 编码（定长补零/score 编入 ID），不在本次范围。

### 进行中

- 无

### 下一步

**安全条线已收完（sec-001 + sec-002）+ P0/P1 + fix-012。**剩余 4 项纯新功能 + 几个小后续：

- `feat-014`：视频转码（需 OSS + 阿里云媒体处理基建）
- `feat-015`：内容全文搜索（需 Canal→Kafka→ES 链路）
- `feat-016`：消息通知系统（内部，无外部基建依赖，推荐优先做）
- `feat-017`：内容标签与话题（feat-018 前置）
- `feat-018`：个性化推荐（依赖 feat-017）
- 小后续：CSP Report-Only → 强制策略（待前端审计）；登录限频 IP 维度；query_hot_feed_zset.lua latest 分支 false-concat 边界

---

## 阻塞 / 风险

- [ ] **`pkg/jwt` 无调用方**：JWT 包目前是死代码；实际登录用 Session（Redis Lua）。fix-001 仅恢复包功能。
- [ ] **评论缓存可能写入空 userName/userAvatar**：UserRpc 失败时缓存仍写入但 user 字段为空。下次读取需有"空则回源补齐"路径；当前读路径行为待确认。
- [x] ~~**热榜同分过滤的 memberId 数字 vs 字典序**~~：fix-012 已修复，改为字符串比较与 Redis 真实排序对齐；回归测试 query_hot_feed_zset_test.go
- [x] ~~**query_hot_feed_zset.lua latest 分支 false 边界**~~：fix-015 已修复，line 32 改为 `if latestId and latestId ~= ""`；新增回归测试 TestQueryHotFeedZSet_LatestMissedFallsThroughToGlobal 覆盖 fall-through 路径
- [ ] **fix-006 关注路径增加了 user-rpc 同步依赖**：user-rpc 故障时关注接口连带不可用。
- [ ] **OSS 未配置**：`deploy/.env` 中 OSS 相关字段为空
- [ ] **视频转码为占位**：`feat-004` 的 `transcode_status` 始终为 TranscodeStatusPending(10)，HLS 播放不可用
- [ ] **无测试覆盖**：项目零测试

---

## 已做决策（本次新增）

- **sec-001 限频仅做 mobile 维度**：LoginReq 没有 client_ip 字段；要做 IP 维度需要改 proto + 前端透传。本轮聚焦"防单账号暴力破解"，IP 维度（防分布式撞库）作为后续独立条目。
- **sec-001 限频默认 5 次/5 分钟**：与业界常见配置一致；零值视为关闭，避免破坏现有空 config 单测。
- **sec-001 限频拒绝时返回独立错误信息**：不与"手机号或密码错误"混淆。理论上会泄漏"某账号当前是否被锁"，但锁本身就是限频后果，可接受；明确提示利于正常用户排错。
- **sec-001 账号禁用路径不计入失败计数**：因为已经通过了密码校验，属于"凭据正确但账号状态异常"，不应触发限频；同时也避免给攻击者额外的状态枚举信号。

- **fix-004 校验错误也返回 200**：用户决策。CustomValidator 本质是"用户输入业务错"，让前端按 body code 走统一处理路径比 HTTP 422 更一致；只有非预期系统错误（default 分支）才升 500，监控/告警能识别真故障。
- **fix-004 middleware 鉴权 401**：网关/前端拦截器靠 status 即可识别"跳登录"，无需解析 body；同时设置 Content-Type: application/json + WriteHeader 在 Write 之前调用，确保 status 真正生效。
- **fix-009 五个子项一次性打包**：feature 描述本身就是"合并修复"，符合"一次提交对应一个 feature_list 条目"。每个子项体量都很小且互不耦合。
- **commentStatusNormal 不一起迁**：只有 `commentStatusDeleted` 跨包使用（8 个文件），`commentStatusNormal` 只在 comment_logic.go 自己用。仅迁需要跨包共享的，最小变更面（rules.md "不扩大范围"）。
- **TranscodeStatusPending 命名**：CLAUDE.md / feat-004 描述里写"占位为 10（未开始）"，故命名 Pending（待开始/排队）。当转码任务接入后可继续扩展 Running / Done / Failed。
- **B-09 顺带改 Name() 返回值**：`Name()` 返回 "CustomePlugin" 字符串，是 GORM Plugin 注册标识。同步改为 "CustomPlugin"，避免外部代码若已按错误拼写访问会断（当前仓库内无依赖此字符串值）。
- **git mv 重命名 SQL 文件**：保留历史关联，对比 add+delete 更利于追踪。

---

## 本次会话修改的文件（session-007，opt-001 user-rpc 旁路缓存）

- `app/rpc/user/internal/common/consts/redis/redis_consts.go` — 新增 `RedisUserInfoPrefix` / `RedisUserInfoMissingSentinel` 常量 + `BuildUserInfoKey(userID)`
- `app/rpc/user/internal/config/config.go` — `Config` 新增 `UserCache UserCacheConfig`（TTL/NegativeTTL/Jitter 四字段，全部带 json default）
- `app/rpc/user/etc/user.yaml` — 新增 `UserCache:` 段（默认值兜底，YAML 可省略）
- `app/rpc/user/internal/common/utils/usercache/cache.go` — 新建。`UserCacheDO` 仅含公开字段（剔除 PasswordHash/Salt/Email）；`Get`/`BatchGet`/`Invalidate`；BatchGet 回写用 `PipelinedCtx`；正负 TTL 均带 jitter；负值哨兵 `"-"`
- `app/rpc/user/internal/common/utils/usercache/cache_test.go` — 新建，10 个 case（4 单查 + 4 批查 + 2 边界）全绿
- `app/rpc/user/internal/logic/userservice/get_user_logic.go` — `userRepo.GetByID` → `usercache.Get`
- `app/rpc/user/internal/logic/userservice/batch_get_user_logic.go` — `userRepo.BatchGetByIDs` → `usercache.BatchGet`
- `app/rpc/user/internal/logic/userservice/get_user_profile_logic.go` — `userRepo.GetByID` → `usercache.Get`
- `feature_list.json` — 新增 `opt-001` 条目 status=done；total 39，done 34
- `progress.md` — 本次记录

**设计要点：**

- 三个接口共用同一 `*do.UserDO` 来源，cache 层统一收敛，不为每个 logic 各写一份缓存逻辑
- `GetMe` 未纳入缓存（敏感字段 + 低频）；`Register` 不预热，懒填即可
- DB miss 写 `"-"` 哨兵，TTL 短（60s），防止穿透；命中哨兵不再回源 DB
- BatchGet 部分命中时仅对 miss 集合调用 `repo.BatchGetByIDs`，结合 `PipelinedCtx` 回写一次 RTT
- Redis 任何错误均降级直接打 DB，只记日志不冒泡——保证可用性优先
- 未引入 singleflight（user 维度分布相对均匀、击穿风险低，等监控显示问题再加）

**受益范围：** feed 推荐/关注流/内容详情/评论填充/关注操作校验 等所有依赖 user-rpc 的读路径，调用方零改动。

---

## 本次会话修改的文件（session-006，fix-015 审计 P2）

- `app/rpc/content/content.go` — C1 xxl-job 启动失败 errors.Is(ctx.Canceled) 跳过、否则 logx.Errorf + os.Exit(1)；imports 增 errors/os
- `app/rpc/interaction/internal/logic/commentservice/batch_get_comments_logic.go` — C2 新增 fillObjCacheTimeout=5s 常量，fillObjCacheBestEffort per-call WithTimeout/cancel
- `app/rpc/content/internal/common/utils/lua/query_hot_feed_zset.lua` — C3 latest 分支用 `latestId and latestId ~= ""` truthiness 判断（防 Lua false-concat）
- `app/rpc/content/internal/common/utils/lua/query_hot_feed_zset_test.go` — 新增 TestQueryHotFeedZSet_LatestMissedFallsThroughToGlobal 回归测试
- `feature_list.json` — 新增 fix-015 条目 status=done
- `progress.md` — 本次记录

## 本次会话修改的文件（session-006，fix-014 审计 P1）

- `app/rpc/interaction/internal/logic/likeservice/unlike_logic.go` — B1 解耦 content-rpc，错误降级 contentUserID=0
- `app/rpc/count/internal/mq/consumer/canal_count_consumer.go` — B2 延迟清理换 bg ctx + 新增 userProfileCacheInvalidateTimeout=5s 常量
- `app/rpc/interaction/internal/logic/commentservice/query_comment_list_logic.go` — B3 cursor/hasMore 用 parseInt64 + 死循环防御
- `app/rpc/interaction/internal/logic/commentservice/query_reply_list_logic.go` — B3 同上对称
- `feature_list.json` — 新增 fix-014 条目 status=done
- `progress.md` — 本次记录

## 本次会话修改的文件（session-006，fix-013 审计 P0）

- `app/rpc/interaction/internal/repositories/favorite_repository.go` — WithResult 用 createErr 捕获并回传
- `app/rpc/interaction/internal/logic/likeservice/like_logic.go` — 新增 likeEventPublishTimeout 常量；publishLikeEvent 加 ctx 参数；goroutine 内 bg+timeout
- `app/rpc/interaction/internal/logic/likeservice/unlike_logic.go` — 同上对称改造；publishCancelLikeEvent 加 ctx
- `app/rpc/interaction/internal/logic/followservice/follow_user_logic.go` — 接线 backfillFollowInboxTimeout（原本死代码常量）
- `app/rpc/interaction/internal/logic/favoriteservice/query_favorite_info_logic.go` — GetCount resp nil 检查
- `feature_list.json` — 新增 fix-013 条目 status=done
- `progress.md` — 审计与本次修复记录

## 本次会话修改的文件（session-006，sec-002）

- `deploy/nginx/nginx.conf` — server_tokens off + 两个 limit_req_zone（api/login）
- `deploy/nginx/conf.d/default.conf` — 5 个安全响应头 + /v1/login、/v1/users 精确匹配套 login zone、/v1/ 套 api zone
- `deploy/nginx/conf.d/ssl.conf.example` — 新建，HTTP→HTTPS 301 + 443 server 块（TLSv1.2/1.3、HSTS、http2 on）
- `deploy/nginx/certs/.gitignore` — 新建，*.crt/*.key/*.pem 永不入库
- `deploy/nginx/README.md` — 新建，启用 HTTPS 5 步流程 + 自签命令 + 验证命令 + 与 sec-001 关系
- `deploy/docker-compose.yml` — nginx 服务加 443 端口与 certs volume（注释，启用时解除）
- `deploy/.env` — 新增 NGINX_HTTPS_PORT=443

## 本次会话修改的文件（session-006，fix-012）

- `app/rpc/content/internal/common/utils/lua/query_hot_feed_zset.lua` — 同分过滤改字符串比较；注释说明字典序对齐
- `app/rpc/content/internal/common/utils/lua/query_hot_feed_zset_test.go` — 新建，miniredis Lua 回归测试
- `feature_list.json` — 新增 fix-012 条目 status=done
- `progress.md` — 本会话第二段记录

## 本次会话修改的文件（session-006，sec-001）

- `app/rpc/user/internal/config/config.go` — 新增 `LoginRateLimitConfig`
- `app/rpc/user/etc/user.yaml` — 新增 `LoginRateLimit.WindowSeconds/MaxAttempts` 默认配置
- `app/rpc/user/internal/common/consts/redis/redis_consts.go` — 新增 `RedisUserLoginFailPrefix` + `BuildUserLoginFailKey`
- `app/rpc/user/internal/common/utils/lua/check_login_rate_limit.lua` — 新建
- `app/rpc/user/internal/common/utils/lua/record_login_failure.lua` — 新建
- `app/rpc/user/internal/common/utils/lua/redis_lua.go` — embed 两个新脚本
- `app/rpc/user/internal/common/utils/ratelimit/login.go` — 新建（CheckLogin / RecordLoginFailure / ClearLoginFailures）
- `app/rpc/user/internal/logic/userservice/login_logic.go` — 集成限频三段调用
- `app/rpc/user/internal/logic/userservice/login_logic_test.go` — 抽出 `newTestLoginLogicWithCfg` + 2 个限频单测

## 本次会话修改的文件（session-005，feat-013 及之前）

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

- [x] 测试通过：`go test ./...`（user-rpc 含 7 个 login case 全部通过）
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