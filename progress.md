# 会话进度日志

## 当前状态

**最后更新：** 2026-07-08
**当前功能：** fix-006 own-feed 迁移 publishbox 修复跨可见性泄露（已完成 · 未提交）

> own-feed（UserPublishFeed）迁到 publishbox.QueryWindow（PUBLIC-only 重建 + 复合游标），并把装配改 publicOnly=true 从读侧堵死私密泄露。删旧锁/重建/内存分页 + 最后一个 query lua 消费者（连带删 QueryUserPublishZSetScript + .lua + parseZSetReply）。`./init.sh` 全绿（32 测试文件）。下一步：Phase C feat-admin-006（用户管理，改 user.proto 升级项）。

---

## 已完成（fix-006 own-feed 迁移 publishbox 修复跨可见性泄露）

**背景**：refactor-006 的 Phase 2 待办。own-feed 冷重建走 `ListPublishedByAuthor`（不过滤 visibility），把 PRIVATE 灌进与大V merge 共享的 `feed:user:publish:{uid}`。

**核实泄露真相（比原描述更准）**：bigV merge 经 follow feed `resolveDetails(ids, true)` 已按 PUBLIC 过滤，**不泄露**；真正泄露在 **own-feed 自身**——它用 `assembleItems(ids, viewerID, false)`（publicOnly=false）把作者私密返回给**任意访问者**；且增量 `writeUserPublishZSet` 无视 visibility 会写私密进 zset，故仅改重建口径不足以消除泄露。

**改动**：
- own-feed 迁到 `publishbox.QueryWindow(authorID, cutoffMillis=0, cursorScore, pageSize)`：复用命中读 / 未命中 PUBLIC-only 重建 / 空哨兵 / DistLocker 防击穿。删 `loadPageIDs`/`queryUserPublishIDs`/`queryUserPublishAllFromDB`/`updateUserPublishCache`/`pageUserPublishRows` + 手搓 RedisLock 重建 + 内存分页 + rebuild 常量 + `contentRepo` 字段 + `buildUserPublishFeedKey`/`userPublishFeedKeepN`。
- 游标 exclusive 纯 millis 串 → 复合 `score:id`，复用 follow feed 的 `parseCursor/afterCursor/formatCursor`，修同毫秒（审核通过 published_at 同批相近）翻页 skip/dup。
- **装配 publicOnly=false → true**：与写扩散 / 大V merge 口径一致，从读侧彻底堵死泄露（即使 zset 残留私密 id 也在装配被过滤）。
- 清死代码：删最后一个 query lua 消费者 → 删 `QueryUserPublishZSetScript` 变量 + `query_user_publish_zset.lua` + `lua_reply.go` 的 `parseZSetReply`（`luaReplyString/Int64` 仍被 recommend feed 用，保留）。

**行为变更（已知）**：作者主页流现**严格 PUBLIC-only**，作者本人也不再经此流看到自己私密内容（项目无 viewer==author 私密展示逻辑，可接受权衡）。`writeUserPublishZSet` 增量写仍可能含私密 id（读侧全 publicOnly=true 已无害；收紧为 PUBLIC-only 属可选 hygiene，未做）。

**验证**：`./init.sh` 全绿（build+vet+test，32 测试文件）。项目未上线无存量游标兼容负担。

---

> Phase B 收官条。发布落待审 + AdminReviewContent（通过/拒绝）+ 进 feed 副作用触发点从发布迁到审核通过 + content 域审核历史表 ran_feed_content_review。用户中途拉起 docker 容器提供 DB（`ENV_FILE=/home/wmr/opt/ran-feed-docker/.env`），故 gorm-gen 正常跑通、审核表落地（未走拆分）。下一步待办：fix-006（own-feed 跨可见性泄露）、Phase C feat-admin-006（用户管理）。`./init.sh` 全绿（33 测试文件）。

---

## 已完成（feat-admin-005 先审后发）

**背景**：Phase B 收官。设计 D5 先审后发——发布落待审、审核通过才进 feed、fanout 触发点从发布时挪到审核通过时。经与用户确认：published_at 取审核通过时间、新建审核历史表、fanout 副作用抽导出函数复用。

**中途转折（DB 可用性）**：起初本地无 `.env`/DB（3306 不通），一度定为"拆分"（先落先审后发、审核表待 DB）。用户随后拉起 docker 容器并给出 `.env` 路径（`/home/wmr/opt/ran-feed-docker/.env`），DB 可达 → 撤销拆分，按完整版落地（含审核历史表 gorm-gen）。

**交付**：
- **发布落待审**：`PublishArticle/PublishVideo` 终态 `PUBLISHED`→`PENDING_REVIEW`，`published_at` 置空，删两份重复 `afterPublish`（连带清 now/time/rediskey 冗余 import）。发布不再触发任何进 feed 副作用。
- **AdminReviewContent**（content.proto 加 `ReviewDecision` + rpc，挂 AdminContentService）：`AdminGetByID` 守卫仅 `PENDING_REVIEW` 可审 → `query.Q.Transaction` 内翻状态 + 落审核记录；通过用 `AdminApproveContent`（状态守卫 + `published_at`=审核时刻 + `updated_by`，affected==0 回滚），拒绝用 `AdminUpdateStatus(REJECTED)`。事务提交后（仅通过）调 `RunPublishFeedEffects` + `contentcache.Invalidate`（遵守事务内不碰 Redis/RPC）。
- **副作用抽取**：`afterPublish` 三件套（publish zset + 热榜脏集合 + follower 扩散）抽成导出 `RunPublishFeedEffects`（fanout_helper.go），发布/审核共用；`admincontentservicelogic` 单向 import `contentservicelogic` 无环。
- **审核历史表**：content 域 `ran_feed_content_review`（content_id/decision/reason/公共字段，`created_by`=审核管理员），DDL 应用本地 MySQL；generator.go 加该表跑 gorm-gen（**容器坑**：该库既有 content/article/video 表注释存为乱码，全表 regen 会污染既有 gen 文件，故仅保留新表两文件 + gen.go 装配，其余 6 个既有 gen `git checkout` 还原；新表注释正确）。`ContentReviewRepository`(WithTx+Create) + `do.ContentReviewDO`，审核记录随状态变更同事务落库。
- **admin-api**：`POST /v1/admin/contents/review`（decision approve/reject + reject_reason），RBAC `content:review`，seed 加权限点并已 apply 本地 DB；待审队列复用 `GET /contents?status=60`。
- **搜索**：走 CDC + `BatchGetContentForIndex`（status=PUBLISHED 过滤），待审自动不进索引，审核通过 CDC 自动补，无需改。

**验证**：`./init.sh` 全绿（build+vet+test，33 测试文件）；单测：buildReviewDO 映射、rbac registry 四内容路由含 review。DB 已应用 review 表 DDL + content 权限 seed。

**C 端连带（已告知用户）**：`GetContentDetail`/own-feed 按 status=PUBLISHED 过滤，作者发布后 C 端暂看不到自己的待审内容；"我的待审列表"属后续产品项，本轮 C 端一行未改。

**遗留**：端到端（起全栈：发布→待审→审核通过进 feed / 拒绝）未现场联调，由单测 + 全绿 + DB DDL 就绪保证。**下一步**：fix-006（own-feed 跨可见性泄露）或 Phase C feat-admin-006（用户管理，改 user.proto 升级项）。

---

## 已完成（fix-009 search 枚举字面量改用 pb 枚举）

**背景**：go-coding skill 新增「常量与枚举」小节后，`search_consts.go` 的 `ContentStatusPublished int32=30` / `ContentVisibilityPublic int32=10` / `UserStatusNormal int32=10` 成为该规则点名的反例——重复定义了 pb 已有的 DB 枚举值，值一改两处漂移。

**改动**：
- `search_content_logic.go` buildQuery：status/visibility filter 改 `int32(content.ContentStatus_PUBLISHED)` / `int32(content.Visibility_PUBLIC)`；换 consts import 为 content pb import。
- `search_user_logic.go`：status filter 改 `int32(user.UserStatus_USER_STATUS_ACTIVE)`（user.proto 已有 UserStatus enum）；换 consts import 为 user pb import。
- `search_consts.go` 删三个 DB 枚举字面量，仅留 `SourceTable*`（canal 源表名，属非 DB 业务字面量，按规则保留）。

**验证**：`./init.sh` 全绿（build+vet+test，32 测试文件）；全仓 grep 无残留引用；值与 pb 定义一致，ES filter 行为不变。

---

## 已完成（feat-admin-004 内容管理）

**背景**：Phase A 地基就绪后进入 Phase B。经与用户确认：本轮只做 feat-admin-004（内容管理，较安全的只读 + 下架/恢复），先审后发（005）留下一轮单独确认；content.proto 的 ContentStatus enum 三值（TAKEN_DOWN/PENDING_REVIEW/REJECTED）一次加齐（同一次 proto 改动，005 不必再动 enum，本轮仅用 TAKEN_DOWN）。

**关键设计验证**：下架 = 状态翻 `TAKEN_DOWN` + `contentcache.Invalidate`。已核实 feed 读路径回源走 `BatchGetPublishedByIDs`（`Status.Eq(PUBLISHED)` 过滤），故下架内容缓存失效后下次 miss 回源即被过滤，**自动从所有流消失，无需清 ZSET**。链路自洽。

**交付**：
- **content.proto**：ContentStatus 加 `TAKEN_DOWN=50/PENDING_REVIEW=60/REJECTED=70`；新增 `AdminContentService`（AdminListContents/AdminGetContentDetail/AdminSetContentStatus）+ 消息，goctl 重生成。`content.go` 注册第三 service（决策 D4 同进程物理隔离，C 端两 service 不改）。
- **content-rpc**：`ContentRepository` 加 `AdminListContents`（可选 status/type/author 筛选 + id 倒序 keyset 游标，仅软删过滤）、`AdminGetByID`（任意状态）、`AdminUpdateStatus`（status + updated_by）。三 logic：列表按类型分组批量取 article/video 标题拼装、满页给 next_cursor；详情复用 GetByContentID 拼正文/封面、含非公开无 viewer 门槛；下架/恢复经纯函数 `validateStatusTransition` 校验状态机后翻状态 + Invalidate。
- **admin-api**：`content.api` 三**静态路由**（`/contents`、`/contents/detail`、`/contents/status` action=takedown/restore）——避开 `:id` 路径参数，因 `AdminRbacMiddleware` 按 `r.URL.Path` 精确匹配。挂 Auth+Rbac+Audit 组，goctl 重生成 routes/swagger；svc/config/yaml 接 `ContentRpcClientConf`（key content.rpc）；operator_id 从 ctx 取 admin_id。
- **RBAC 激活**：registry 登记 `content:list/detail/takedown`——**Phase A 门禁首次真正生效**；`seed_content_permissions.sql` 幂等播种三权限点 + super 补绑 content 模块。
- **单测**：`validateStatusTransition` 状态机 10 例、`buildAdminContentItem` 映射、`normalizePageSize` 边界、`optional*` 零值转 nil；rbac registry 断言三内容路由。

**验证**：`./init.sh` 全绿（build+vet+test，31 测试文件）。

**遗留**：端到端（起全栈 admin-rpc+content-rpc，下架后 feed 消失、RBAC 拒绝无权限角色）未现场联调，由单测 + 全绿保证；`seed_content_permissions.sql` 待部署时执行一次。**下一步 feat-admin-005**：先审后发（发布落待审 + AdminReviewContent + fanout 触发点从发布迁到审核通过），改发布/fanout 写路径属升级项，动手前需与用户确认。

**未提交**：feat-admin-004 全部改动（content.proto + content-rpc + admin-api + seed sql）。

---

## 已完成（feat-admin-003 RBAC + 审计埋点）

**背景**：Phase A 地基收尾条。Phase A 尚无业务门禁路由 故本条交付 RBAC/审计**执行基础设施**（live 门禁随 Phase B 业务路由激活），审计中间件当下即生效（登出被记录）。未改 admin.proto 无升级。

**交付**：
- `internal/common/rbac` 包：`registry`（route→权限点 map，Phase A 空，Phase B/C 路由登记所需 code）+ `HasPermission` 纯校验 + `LoadPermissions`（Redis 权限缓存 cache-aside，回源 ListAdminPermissions，哨兵 `__loaded__` 区分零权限与未命中）+ `Invalidate`（变更失效）。
- `AdminRbacMiddleware`：按 registry 取路由所需 code，未登记只需登录，缺权限返 `100203 无操作权限`。
- `AdminAuditMiddleware`：已登录非 GET 请求，`statusWriter` 捕获状态码，`threading.GoSafe`+bg ctx 异步落 `operation_log`（action=METHOD+path / ip / result）。
- 两中间件挂 protected 组（Auth→Rbac→Audit），goctl 重生成 routes；svc 注入。
- 单测：registry 查找、HasPermission、toSet 剔哨兵、LoadPermissions（miss 回源 / hit 不回源 / 零权限哨兵 / Invalidate 重载 / loader 错误）。

**冒烟**（真实 admin-rpc:5008 + admin-api:5010）：登录→登出后 `operation_log` 落一行 `admin_id=1 action='POST /v1/admin/logout' result=200`（冒烟数据已清）。

**Phase A 收尾**：admin-rpc 骨架 + admin-api 登录闭环 + RBAC/审计 全部就绪。**下一步 Phase B**：content-rpc 新起 `AdminContentService`（列表/下架）+ admin-api 内容管理 + 先审后发（改发布/fanout 写路径）——首次动 content.proto，属升级项，动手前需与用户确认。

**未提交**：feat-admin-001/002/003 全部改动 + 会话前的 content/service_context.go。

---

## 已完成（feat-admin-002 admin-api 登录闭环）

**背景**：用户要先定义 admin BFF 接口供前端并行开发 → 契约先行 随后补全逻辑做到端到端可登录。

**交付**：
- 新建 `app/admin`（HTTP :5010 纯 BFF 与 app/front 平级）：契约 `POST /v1/admin/login`、`POST /logout`、`GET /me`（后两者挂 `AdminAuthMiddleware`）；响应带 **permissions[]** 供前端 RBAC 驱动菜单免返工；swagger 出 `app/admin/swagger/admin.json`。
- 鉴权：`AdminAuthMiddleware` 用 go-zero 单命令（GetCtx 校验 + ExpireCtx 滑动续期），`admin:session:{token}` 独立命名空间与 C 端隔离；登录顶掉旧 token。
- admin-rpc 4 logic 由桩转实现：AuthenticateAdmin（bcrypt `hash=pwd+salt` 校验 + 状态）、GetAdmin、ListAdminPermissions（admin→角色→权限 code 去重）、WriteOperationLog。
- 超管播种 `script/sql/ran-feed/admin/seed_super_admin.sql`（幂等）：super 角色 + 5 权限点 + 全绑定 + 账号 **admin / Admin@123456**（bcrypt）。已建 admin id=1。

**端到端冒烟**（真实 admin-rpc:5008 + admin-api:5010 + etcd/redis/mysql）：错误密码拒 / 正确密码返 token+5 权限点 / 无 token /me 返 100201 / 带 token /me 返 admin_info+权限 / 登出后 /me 失效 —— 全部符合预期。

**下一步 feat-admin-003**：RBAC 路由级权限点校验中间件 + 操作审计自动埋点（写操作落 operation_log）+ 权限点按业务模块扩充。

**未提交**：feat-admin-001 + feat-admin-002 全部改动 + 会话前的 content/service_context.go。

---

## 已完成（feat-admin-001 admin-rpc 骨架）

**背景**：项目原为纯 C 端 B 端 0 代码。经讨论定 admin 落地方案(见 `ADMIN-DESIGN.md`):独立 admin-rpc 管理域 + admin-api 纯 BFF + 各域新起 `AdminXxxService` 承接 admin 方法(不违反 refactor-007 每域独占表不变式)+ Redis session 鉴权 + 先审后发。

**改动（纯地基 无业务逻辑）**：
- `admin.proto`：`AdminService` 4 方法(AuthenticateAdmin/GetAdmin/ListAdminPermissions/WriteOperationLog)goctl 生成骨架 logic 桩返回空;main 补 `envx.Load`+`conf.UseEnv`+`ServerGrpcInterceptor` 对齐 user-rpc。
- 6 表 DDL 于 `script/sql/ran-feed/admin/`：admin_user/role/permission/user_role/role_permission/operation_log 沿用公共字段 已建于本地 MySQL。
- gorm-gen 出 6 model+query;5 Repository 全软删过滤;svc 挂 MysqlDb+query.SetDefault 仅 MySQL(无 Redis/Kafka/xxljob)。
- **端口修正**：admin-rpc 取 **5008**(5005/5007 被 content/search 的 xxl-job 执行器占用 见 ran-feed-docker/.env);Prometheus 9297。`ADMIN-DESIGN.md` 端口表已按真实 .env 更正。

**验证**：`./init.sh` 全绿(build+vet+test 30 测试文件)。

## 下一步

1. 工作区未提交:本条(feat-admin-001)+ 会话前的 service_context.go 改动 待提交
2. feat-admin-002 admin-api BFF 骨架 + 登录/登出 + AdminAuthMiddleware(依赖 admin-rpc AuthenticateAdmin 落地)
3. feat-admin-003 RBAC 中间件 + 权限点播种 + 操作审计埋点

**遗留提示**：admin-rpc 的 4 个 logic 目前是空桩 真实逻辑在 feat-admin-002/003 填(登录校验落 AuthenticateAdmin/RBAC 落 ListAdminPermissions/审计落 WriteOperationLog)。

---

## 已完成（热榜定时任务可观测性）

**背景**：冷/快更执行失败时只看到 runner 一行裸 error 无 traceId 无阶段 定位困难。

**改动**：
- `pkg/xxljob/runner.go`：每次执行在协程内起根 span（server 端）→ Telemetry 已配 `Sampler:1.0` 故 traceId 有效，begin/fail/finish/callback 全链串联，下游 DB/RPC span 正确嵌套；裸 `go func()` 换 `threading.GoSafe`（内层原 recover+回调保留）。
- `hot_fast_update`（job+recompute）与 `hot_cold_update`：统一 `logx.WithContext(ctx)`（run ctx traceId 才生效），关键节点补中文日志（开始/让路/放弃/条数/快照重建或跳过/完成），每个 `return "", err` 与跨边界失败点用中文阶段名包裹。
- `content.yaml`：`XxlJob.HTTPTimeout` 10s→60s。
- 用户后续把两处锁释放由 `ReleaseCtx(ctx)` 改回 `Release()` 避免 ctx 取消导致解锁失败。

**验证**：`./init.sh` 全绿。

---

## 已完成（feat-007 搜索自动补全）

**背景**：`SEARCH-DESIGN.md §1.1` 原把自动补全列为非目标,现补上。补内容标题 + 用户昵称两类,支持中文字符与拼音前缀。

**方案（completion suggester + 复用现有索引/CDC，零新管道）**：
- 两索引各加 completion 字段:`ran-feed-content.title_suggest`、`ran-feed-user.nickname_suggest`,存整条标题/昵称,weight 内容取 `hot_score`(float→int clamp)、用户暂 0。
- mapping `settings.analysis` 定义 **pinyin_suggest analyzer**(pinyin tokenizer 开 full/joined/first_letter/original),completion 字段用它 → 中文字符与拼音前缀同时命中。suggest 字段并入 `_source.excludes`。
- 写入零新管道:`document.go` 映射器建文档时顺带填 suggest(空文本返 nil 免 ES 拒空 input);跑现成 reindex 回填。
- **统一入口、结果分流**:`/v1/search/suggest` 并行查两索引 → 合并成带 `type`(content/user) 的建议;点击用建议词跑普通搜索、落对应 tab。不做跨类型混合排序(不碰既定非目标)。
- 稳定性:单索引 suggest 失败仅该组为空、不整体报错(按键接口要稳)。

**改动**:`search.proto` 加 `SuggestType`+`Suggest` RPC;`es/suggest.go` 新增 `Suggest`(+ 可测的 `parseSuggestOptions`);`document.go` 加 `*completionInput` 字段与填充;`suggest_logic.go`(rpc)并行合并;front `search.api` 加 `/suggest` + `suggest_logic` 透传映射 type。

**验证**:`./init.sh` 全绿(29 测试文件)。单测:`parseSuggestOptions` 正常/空/坏 JSON、document suggest 填充 + weight 取整/负归零/空标题省略。

**部署前置(端到端拼音)**:装 `analysis-pinyin`(ran-feed-docker,用户负责)→ 删旧索引 → 重启 search-rpc(`EnsureIndices` 是"不存在才建",故必须先删)→ reindex 回填。

---

## 上一状态（refactor-008 搜索游标分页，done）

---

## 已完成（refactor-008 搜索游标分页）

**背景**：搜索原用 `from/size` 偏移分页,前端是 feed 式无限下拉——用不上跳页,却吃两个代价:①ES `max_result_window` 默认 1 万,`from+size` 越界**直接报错**(下拉滚够久必崩);②深分页每 shard 取 `from+size` 归并丢弃越深越慢。且原 sort `[_score,hot_score,published_at]` 无唯一 tiebreaker,页边界可能漏/重。

**方案(游标 = ES 自己的 sort 数组原样回传)**:
- 游标即上页末条命中的 ES `sort` 值数组 → `json+base64` 成不透明串给前端 → 回传解码后原样作 `search_after`。search-rpc **不理解排序字段语义**,只做黑盒往返。
- sort 末位补唯一键:content 加 `content_id asc`、user 加 `user_id asc`(均 keyword 可排序),消除同分页边界不稳。
- `buildQuery` 删 `from`;`cursor` 非空时加 `search_after`。
- proto `SearchContentReq/SearchUserReq` 的 `page` → `cursor`,响应加 `next_cursor`(空=无下一页,满页才给);front `.api` 同步。
- **保留 `total`**(ES 默认 track_total_hits 上限 1 万 best-effort,零成本);**放弃跳页**、不引入 PIT(接受与既有 feed 同级翻页漂移)。

**改动**:`es.Hit` 加 `Sort []any` 并解析;`paging.go` → `normalizeSize` + `encode/decodeCursor` + `nextCursor`;两 logic 的 `buildQuery` 与出参;front 两 logic 透传 `cursor/next_cursor`。history 三接口、ES mapping、写入链路不动。

**验证**:`./init.sh` 全绿(28 测试文件)。新增单测:游标混合类型往返、空/非法游标、`nextCursor` 满页/到底/空命中、`buildQuery` 无 from + tiebreaker + search_after 加入/省略、`normalizeSize` 边界。非法游标返回错误(不静默从头,防翻页死循环)。

**遗留**:端到端(真实 ES 下拉多页串联无漏重、翻过 1 万条不报 `max_result_window`)未现场联调,由单测 + 全绿保证。

---

## 上一状态（refactor-007 search 去跨域直读 改 RPC 回源，done）

---

## 已完成（refactor-007 search 去跨域直读 改 RPC 回源）

**背景**：search-rpc 原是全项目唯一直读 4 张别域表（content/article/video/user）的服务，仅因开发期共库能跑；目标架构 database-per-service 下拆库即断。查询/富化链路本就走 RPC，问题只在写入/建索引两条链路的「回源读表」。

**方案（分 3 Phase）**：
- **A · content-rpc**：`FeedService` 加 `BatchGetContentForIndex(content_ids)` / `ListContentForIndex(cursor,limit)`，只返回**可索引**（已发布+公开+未删除）内容的原始投影（title/description/body/hot_score/published_at/version=updated_at 毫秒）；判活与三表组装归 content 域。
- **B · user-rpc**：`UserService` 加 `BatchGetUserForIndex` / `ListUserForIndex`，只返回正常+未删除用户投影。
- **C · search-rpc**：消费者 `handleContent/handleUser` 改调 RPC，返回项 upsert、请求了但未返回的 id 判 delete（新 `missingIDs`）；reindex job 改 `ListXxxForIndex` 游标循环；`document.go` 由回源 `Assembler` 改**纯映射器**；canal 路由改本地 `SourceTable*` 常量；svc/config/yaml 接 `ContentRpc(feedservice)`/`UserRpc` 客户端；**删** 4 个跨域 repo + 8 个 `.gen.go`，trim `query/gen.go`，`generator.go` 收到 2 表。

**判定归属**：可索引判定从 search 硬编码 `status=30/visibility=10` 彻底移进源域 RPC，`classifyContent/classifyUser` 删除。
**解耦深度**：CDC 保留作「变更触发器」，search 只留源表名字符串常量路由 canal 事件。

**连带点**：goctl 1.10.1 regen 只再导出 req/res 别名、丢掉嵌套类型别名 → user client 的 `UserInfo/UserProfile` 被删导致 content/interaction 编译失败；已按原样补回，使 regen 净效果仅新增 ForIndex 别名。

**验证**：`./init.sh` build+vet+test 全绿（28 测试文件）。新增单测：content/user 投影映射、search `missingIDs` 分区、`document.go` 映射器。全仓 grep 无对已删 model/repo 的残留引用。

**遗留**：端到端（真实 ES + Canal + RPC 回源）未现场联调，需三服务 + ES + Canal 起全栈；本次由单测 + 全绿保证。GORM model 因本地无 `.env`/DB 凭据未用 `go run ./gen/generator.go` 重跑，改为手工 trim 生成物 + 同步 `generator.go`（下次带 DB regen 可复现同一结果）。

---

## 上一状态（feat-006 搜索端到端联调 + 架构文档，done）

## 已完成（feat-006 搜索端到端联调 + 架构文档）

搜索 P6。真实 ES(8.12.2+IK)+Canal(search destination)+Kafka 全链路联调。

**验证结果（grpcurl 直连 search-rpc + MySQL/ES 观测）**：
- CDC 增量：touch 行 → 约 4s 进 ES（content5 + user4）。
- 查询：`测试` 5 命中按 score 排序，title+description 双 IK 高亮。
- content_type tab：图文 5 / 视频 0；用户搜索：哈哈→1、ran→2。
- 历史：Record / List / 去重提前 / 删单条 / 清空 全对。
- 删除传播：软删 content → 约 2s 移出索引、搜索排除。

**联调中发现并修复 feat-004 一个 bug**：删后**快速恢复**的内容不重新可搜——delete 用 canal 事件 ts、upsert 用行 updated_at，两个时钟源不一致 → restore 的 version 被 delete tombstone 以 ES 409 挡住，直到下次 reindex 才恢复。修法：`writeES` 增量路径 upsert/delete **统一用 canal 事件 ts 作 version**（单分区内单调，canal ts ≥ 行 updated_at 与 reindex 仍一致）。重启 search-rpc 复验：删→移出、恢复→约 4s 回插，正确。

**文档**：`docs/architecture.md` 拓扑加 search-rpc(:5006) + 两条 CDC/重建链路，新增「搜索架构」节。按用户口径**不单独迁 search 设计文档**，`SEARCH-DESIGN.md` 暂留由用户自行处理。

**ran-feed-docker**：加 `kafka-init` 服务预创建 `ran-feed-{count,search}-canal` topic，免消费者在 topic 不存在时订阅拿到 0 分区（联调时踩到过：search-rpc 先于 topic 启动导致 0 分区，重启才恢复）。

`./init.sh` 全绿（27 测试文件）。front 富化整链未起全栈、未现场联调，由单测 + `BatchGetContentItems` 复用 resolver 保证。

## 已完成（feat-005 front 搜索接口 + 富化 + 历史记录）

搜索 P5。front 网关接搜索，复用 feed 拼装链富化。

- **content-rpc 升级点**：content.proto FeedService 加 `BatchGetContentItems(content_ids, viewer_id)`，goctl 重生成；logic 薄封装直接调内部 `resolver.assembleItems(ids, viewerID, publicOnly=true)`——**真复用 feed 拼装链**（之前富化链封在 content-rpc 内、front 够不着，故开此口）。
- **front 接口**：`search.api` 定义 `/v1/search/content`（带 content_type tab）`/user` `/history`(GET/DELETE)，挂 OptionalLoginMiddleware；front.api import；goctl api go + swagger 重生成。
- **svcCtx**：接 `SearchRpc`（config + front-api.yaml `SearchRpcClientConf`，key search.rpc）。
- **四 logic**：
  - SearchContent：search-rpc 出 ranked content_ids + 高亮 → `FeedRpc.BatchGetContentItems` 富化 → `assembleSearchContentItems` 按 content-rpc 顺序合并高亮。
  - SearchUser：`UserRpc.BatchGetUser` + 并行 `FollowRpc.GetFollowSummary`（is_followed + 粉丝数），按搜索序拼装。
  - ListHistory / DeleteHistory：调 search-rpc，匿名守卫（list 返回空 / delete 直接成功）。
  - 隐式记录：`recordSearchHistory` 异步脱离请求 ctx + 3s timeout，非致命（go-coding 副作用 ctx 规范）。

单测覆盖高亮合并 + 保序 + nil 跳过。`./init.sh` 全绿（27 测试文件）。**运行期遗留**：端到端验证留 feat-006，需 P0 canal search destination 灌数据。

## 已完成（feat-004 CanalSearchConsumer 增量同步）

搜索 P2。订阅 canal CDC 增量维护 ES，复用 feat-002 的回源组装链。

- **去重**：gen `ran_feed_mq_consume_dedup` + `MqConsumeDedupRepository.InsertIfAbsent`（consumer=`search.canal_consumer`），照搬 count 幂等闸。
- **解析**：`canal_message.go`（adapt count，内联 `parseInt64` 兼容 canal 的字符串列值）。
- **消费者**：`CanalSearchConsumer` 按表路由——content/article/video 取 `content_id`、user 取 `id`；逐行去重后 `GetByIDs`(is_deleted=0) 分类：命中且 `status=30&visibility=10`（用户 `status=10`）→ upsert，软删/下架/转私密/缺失 → delete。复用 `indexer` 组装。
- **删除**：`es.BulkDelete` external version（取事件 ts 毫秒）忽略 404/409；写 ES 非致命，靠重建补。
- **装配**：`init_consumer.Consumers` 接 kq；config/yaml 加 `KqConsumerConf`（topic `ran-feed-search-canal`）；main serviceGroup 注册消费者。

单测覆盖 classifyContent/classifyUser（upsert/草稿/私密/封禁/缺失）、parseInt64、rowEventID 稳定性。`./init.sh` 全绿（26 测试文件）。

**ES 已就绪**：ran-feed-docker 已起 ES + IK 8.12.2（只起 ES 容器），ik_smart 分词验证通过；docker 仓库配套 install-ik.sh/gitignore/.env(.example)/README 已补，**未提交**。
**运行期遗留**：feat-004 联调需在 canal 加 `search` destination（订阅 content/article/video/user → topic `ran-feed-search-canal`）——canal 现仅 count 实例。

## 已完成（feat-003 SearchService 查询逻辑与历史记录方法）

搜索 P4。填 feat-001 的 5 个 logic 桩。

- **es.Search** `internal/es/search.go`：执行 `client.Search` 并解析 hits（_id/_score/highlight），`Hit.FirstHighlight` 取首片段。search-rpc 只返回 ranked IDs+score+高亮，展示字段留 front 富化。
- **SearchContent**：`bool{must: multi_match(title^3/description^2/body, ik_smart), filter: status=30/visibility=10/is_deleted=0 + 可选 content_type}`，`sort[_score, hot_score, published_at]`，highlight title/description；_id 解析回 content_id。
- **SearchUser**：`multi_match(nickname^3/bio)` + filter status=10/is_deleted=0，sort[_score]。
- **分页** `paging.go`：`pageToFromSize` 默认 10、上限 50、page 从 1。
- **历史三方法**：RecordHistory/ListHistory/DeleteHistory 调 `SearchHistoryRepository`（匿名/空词守卫，all 清空否则删单条），repo 作 logic 字段照搬 count。
- **降级**：ES 查询失败 `errorx.Wrap` 上抛，由 front 返回空+错误码降级（设计 §4.5）。

单测覆盖分页边界、content 查询 DSL（含 content_type 可选分支）、user 查询 DSL、highlight 解析。`./init.sh` 全绿（25 测试文件）。**运行期遗留**：真实查询验证待 P0 起 ES 且 feat-002 灌入数据。

## 已完成（feat-002 SearchReindexJob 全量回填/周期重建）

搜索 P3。xxl-job 全量扫 MySQL 以真相源重灌 ES，初始灌入 + 周期兜底漂移。先于增量(feat-004)做，让 ES 有数据可供后续查询 Phase 验证。

- **读模型/仓储**：gorm-gen 加 content/article/video/user 四张读表；`ContentRepository.ScanPublishable`（已发布+公开+未删除，id 游标分批 500）、`UserRepository.ScanActive`（正常+未删除）、`Article/Video.GetByContentIDs`（软删过滤）。状态值提常量 `internal/common/consts/search_consts.go`，软删复用 `pkg/enum`。
- **文档组装** `internal/logic/indexer`：ContentDoc/UserDoc；`Assembler.AssembleContentDocs` 回源 article/video 拼整篇（id 转 keyword 字符串、published_at 毫秒、文章取标题简介正文、视频仅标题），`AssembleUserDocs`；version 取源行 updated_at 毫秒。**feat-004 消费者将复用此组装链**。表驱动单测覆盖文章/视频/nil 指针/空输入/用户。
- **ES bulk** `internal/es/bulk.go`：`esutil.BulkIndexer` 封装 `BulkUpsert`，external version 防乱序，409 冲突视为已有更新文档不计失败。（BulkDelete 留 feat-004 用到再加。）
- **job/装配**：`internal/cron/search_reindex` 游标全量扫 → 组装 → bulk 灌 content/user；`cron.go` Register；main 接 xxl-job executor（照搬 count），config/yaml 加 XxlJob 段。

`./init.sh` 全绿（23 测试文件）。**运行期遗留**：真正灌库要等 P0 起 ES+IK，并在 xxl-job admin 配 handler `search.reindex` 手动触发一次。

## 已完成（feat-001 search-rpc 服务骨架）

按 SEARCH-DESIGN.md 排定的 Phase 开搭搜索。本条只搭骨架与建表 不含业务逻辑（查询/同步留 feat-002~005）。

- **proto + 骨架**：新建 `app/rpc/search/proto/search.proto` 定义 SearchService 五方法（SearchContent/SearchUser/RecordHistory/ListHistory/DeleteHistory）`goctl rpc protoc --style=go_zero --multiple` 生成 pb/grpc/client/logic 桩/server/main。logic 桩返回空待后续填。
- **ES 客户端**：go.mod 引入 `go-elasticsearch/v8 v8.12.1`（对齐 ES 8.12.2）；`internal/es/` 封装 MustNewClient 与 EnsureIndices（content title^3/description^2/body ik_max_word body 不入 _source；user nickname ik+keyword/bio）幂等建索引；接入 svcCtx。
- **历史记录表**：`script/sql/ran-feed/search/ran_feed_search_history.sql` 对齐项目公共字段（status/version/is_deleted/created_by/updated_by + UNIQUE(user_id,keyword) 去重）已应用到本地 MySQL；`go run ./gen/generator.go` 生成 model+query；`SearchHistoryRepository`（Upsert/ListRecent/DeleteOne/Clear 软删过滤）。
- **main**：仅装 gRPC server + 启动调 EnsureIndices（ES 不可用非致命 log）；consumer/cron 留 feat-004/feat-002。

`./init.sh` 全绿（build+vet+test）。**运行期遗留**：实际建 ES 索引需 P0 把 ES+IK 插件起起来（本机 9200 未起 IK 未装）；search.yaml 引用的 `ES_ADDRESS/ES_USERNAME/ES_PASSWORD` 待 P0 在 ran-feed-docker `.env` 补上。

## 已完成（fix-007 互动查询接口补 OptionalLoginMiddleware）

本地起服务实测业务正确性时发现:登录用户调 like/info 永远 is_liked=false。根因 interaction 读组(like/info、like/info/batch、favorite/info、comment/list、comment/reply/list)的 .api @server 块漏了 middleware 注解,生成的 routes.go 没挂中间件 → token 不解析 → userId=0 → queryIsLiked 的 `if userID<=0 return false` 短路。修法在 interaction.api 读组补 `middleware: OptionalLoginMiddleware`,goctl --style=go_zero 重新生成。./init.sh 全绿,运行时实测 is_liked 由 false 变 true、匿名仍 false 不报错。提交 ccf5270。

## 已完成（fix-008 补 content FollowFanOut.DeadlineWindowDays 配置）

content.yaml 的 FollowFanOut 段缺 DeadlineWindowDays,而 config.go 及 publishbox/feed/fanout 十余处都读它且无代码默认 → 运行时为 0,关注流读窗口塌缩为 0 天。补 `DeadlineWindowDays: 14`。提交 faa255d。

> 本地联调备忘(非本仓库改动):部署配置迁到 ran-feed-docker 后,宿主机 go run 需 `export ENV_FILE=指向 docker .env`(host 改 127.0.0.1)、`/etc/hosts` 加 `127.0.0.1 kafka etcd mysql redis xxl-job-admin`(kafka/etcd advertised 名宿主机需可解析);canal 1.1.7/1.1.8 缺 `canal.instance.global.lazy` 会启动 NPE,已在 docker 仓库补上。

---

## 已完成（chore 部署配置迁出到 ran-feed-docker）

部署层独立到 [ran-feed-docker](https://github.com/mmfiRan/ran-feed-docker) 仓库，本仓库只留源码与镜像构建管线。./init.sh 全绿。

- **删除 `deploy/`**：docker-compose 与 nginx/grafana/prometheus/otel/mysql/canal/filebeat/logstash 配置、`.env`、前端镜像 tar 共 18 个跟踪文件，已迁到部署仓库。
- **保留镜像构建**：`build/*.Dockerfile` + `.github/workflows/docker-publish.yml` 留在源码仓库，CI 构建推 ghcr.io，部署仓库拉取消费。
- **保留 SQL**：`script/sql/` 作为 schema 真相源留下，部署仓库 `mysql/sql/` 为同步副本。
- **文档同步**：README/README_EN 去掉 `deploy/` 死链改指部署仓库，CLAUDE.md 的 `.env` 参考改为部署仓库 `.env.example`。

---

## 已完成（refactor-006 发件箱查询抽象 publishbox · Phase 1）

把 publish zset(发件箱 `feed:user:publish:{uid}`)的查询从读路径抽出来，新建 `app/rpc/content/internal/logic/publishbox`。./init.sh 全绿。

- **新增 `publishbox.QueryWindow`**：命中直接读 / 未命中回源重建 / 空作者写哨兵防穿透。读法沿用复合游标——裸 `ZrevrangebyscoreWithScoresByFloatAndLimit` maxScore **inclusive**，边界同分交 `mergeScored` 按 id 过滤（不能套 own-feed 的 exclusive 游标 query lua，会漏边界项）；空结果再 `EXISTS` 区分 cache miss 与窗口内真空。
- **防击穿改用公共方法**：`cache.DoWithLock`(DistLocker) 取代手搓 `RedisLock`+`time.Sleep`。大V发件箱是跨 pod 热点，全集群只放一个重建 + 拿锁后双检 + 未拿锁轮询等待复用；svcCtx 挂 `PublishBoxRebuildLocker = cache.NewDistLocker(redisClient)`，对齐 interaction 的 `LikeUserRebuildLocker`。`ErrLockBusy` 本轮该作者降级缺席，不阻塞整流。
- **重建脱离请求 ctx**：`context.WithoutCancel`，回源走 `ListPublishedByAuthorWithinWindow(0, PUBLIC-only)` 与 fanout 写口径一致，写满或写哨兵后回读本页。
- **bigV merge 切换**：`fetchBigVContentIDs` 改调 `box.QueryWindow`，删裸读的 `queryBigVPublishIDs` —— **修掉大V发件箱 TTL 过期后内容从所有关注者 feed 永久消失的假空当真空 bug**。
- 写侧(fanout/发布)与 own-feed 本次未动。

## 遗留风险 / 下一步（fix-006 · Phase 2 待办 not-started）

- **跨可见性泄露**：own-feed 的 `loadPageIDs` 冷重建走 `ListPublishedByAuthor`（**不过滤 visibility**），会把 PRIVATE 灌进和大V merge 共享的同一个 `feed:user:publish:{uid}` key，bigV merge 读同 key 喂给关注者 → 私密内容可能泄露。修法：把 own-feed 迁到 `publishbox.QueryWindow`(cutoffMillis=0 全量) 复用 PUBLIC-only 重建，一并删掉 `user_publish_feed_logic` 里重复的锁+重建+内存分页。注意 own-feed 现为 exclusive 字符串游标，迁移后改复合游标需回归翻页行为。

---

## 已完成（refactor-005 关注流推拉模型加固）

对照 `docs/follow-feed-design.md` 共识方案，在 refactor-004 上加固。各步 ./init.sh 全绿。

- **大 V 改 sticky 落库**：新增 count `ran_feed_big_v` 表(只增不删 真相源)+BigVRepository；CDC `syncBigVMember` 跨阈值只 INSERT IGNORE+SADD 去掉 SREM 降级分支，消除降级空档与阈值抖动；阈值提到 50000。
- **重建任务迁 xxl-job**：`time.Ticker` 自循环改 xxl-job 触发的 `BigVReconcileJob`(internal/cron/bigv_reconcile)：复查 count_value≥阈值补漏晋升 + 以表为真相源 RENAME 重灌全局集合兜底 Redis 丢失；count.go 接 executor+XxlJob 配置，删旧 internal/job。
- **inbox 口径统一只装小号**：重建/冷兜底用 `loadGlobalBigVSet` 过滤大 V(`filterSmallFollowees`)，大 V 永远读时 merge 不进 inbox。
- **读路径冷热统一**：FollowFeed 结果=merge(inbox 来源, 大V publish)，大 V merge 永远执行消除只关注大V用户首屏空白；异步重建+coldBackfill 合并为同步 `buildInboxSync`(一次查询既填缓存又出首屏 删 goroutine/锁)；空结果写不可见哨兵(id<=0)负缓存。
- **复合游标 score:id**：`mergeScored`/`pageScoredFromRows` 按(score,id)双键修同毫秒翻页 skip/重复，兼容旧纯 millis 游标。
- **查询去 Lua**：inbox 与大 V publish 查询退成原生 `ZrevrangebyscoreWithScoresByFloatAndLimit`+`Exists`+`Expire`，删 `query_follow_inbox_zset.lua`；backfill lua 去末尾 ZSCORE 循环改累加 ZADD 返回值；update lua 因 rebuild 批量大保留循环。
- **go-zero v1.10.2**：用其 `DoCtx` 通用命令口，大 V 修正 RENAME 由 Eval 改原生 DoCtx。
- **后续**：作者主页流 user_publish_feed 仍用 publish 查询 Lua，待统一。

---

## 已完成（refactor-004 关注流 deadline 时间窗口重构）

分 5 个 Phase 落地 各自 ./init.sh 全绿：

- **Phase 1 大 V 模型 → 全局集合 feed:bigv:global**：count CanalCountConsumer.reconcileBigVSet 按 FOLLOWED 阈值 SADD/SREM 增量维护；新增 count internal/job/BigVRebuildJob 启动预热+每小时全量重建(临时 key SADD 后 RENAME 原子切换)兜底漂移与 Redis 丢失；content 写扩散 fanout 与读路径 bigv_merge 改 SISMEMBER/SMEMBERS∩关注 去 count-rpc 扫描与 500 截断漏读 cap 提 5000。Part B 认证号业务标记需加 schema 字段 后置未做。
- **Phase 2 deadline 地基**：inbox/publish zset score content_id→published_at 毫秒；写 lua 加 ZREMRANGEBYSCORE 窗口裁剪+EXPIRE；query lua 游标改时间戳+min=cutoff+WITHSCORES 返回(member,score)；新增 followwindow(CutoffMillis/TTLSeconds/WriteArgs) 与 lua_reply.parseZSetReply/scoredID/mergeScored 按 published_at 归并；冷路径 ListFollowByAuthorsCursor 与 user_publish 全改 published_at 游标。inbox 窗口裁剪 publish 不裁剪(承载 user_publish 全量历史 窗口只读时施加)。
- **Phase 3 关注 backfill**：FollowUser 翻转判定只在 unfollow→follow 触发；content BackfillFollowInbox 失效 viewer bigv 缓存+大 V 跳过+窗口对齐(冷则回源 DB ListPublishedByAuthorWithinWindow 取 PUBLIC)。
- **Phase 4 取关清理**：新增 RPC PurgeFolloweeFromInbox(改 proto goctl 生成) 失效 bigv 缓存+大 V 跳过+ZREM followee 窗口内 content_id；UnfollowUser 翻转判定+异步调用。
- **Phase 5 收尾**：listFollowees 三合一 listFolloweesCapped；backfill/purge/fanout 取数与大 V 判定去重收口 follow_inbox_source(loadFolloweeWindowContent/isBigVAuthor)。
- 项目未上线 不考虑存量迁移。

---

## 已完成（refactor-003 feed 二级缓存内容详情统一）

- 设计 L1 排序 ZSET 各 feed 自管不动 新增 L2 按 content_id 缓存内容本征详情 与观察者无关 author 名头像走 user usercache like 读时旁挂不缓存
- 模型 do.ContentDetailDO 六字段加 visibility 存储 String 加 JSON 加 MGET key content:detail:{id} 与 usercache 同构
- 新增 contentcache 包 BatchGet(ids loader) miss==过期单分支 批量回源 TTL 叠 jitter 负哨兵防穿透 只降级不阻断 不做击穿防护 L2 重建廉价 锁留给 L1 配 ContentCacheConfig 走 default 不写 yaml
- 共享 contentDetailResolver loadDetails 回源 content 行加 brief resolveDetails 加 publicOnly 过滤 loadAuthorsAndLikes 并行查作者点赞 assembleItems 出 ContentItem
- 四条 feed 接入各删一份重复 buildBriefMaps 加 buildUserAndLikeMaps 加 buildItems recommend/follow 读时过滤 PUBLIC follow 因返回 FollowFeedItem 用 resolveDetails 加 buildFollowItems 自组装 冷备份路径改走 ids 经 L2
- delete_content 写路径接 contentcache.Invalidate
- contentcache 单测全绿 ./init.sh 通过

---

## 已完成（refactor-002 收藏链路去 Lua 化）

- interaction 侧（已提交 bf5ca5a）删 favorite:rel 关系缓存与负缓存 QueryFavoriteInfo 直接回源 DB Favorite/RemoveFavorite 纯 cache-aside
- content 侧本会话完成 收藏流 L1 改热头部旁路缓存
  - 只缓存最新 capacity=300 条 TTL=24h 页在头部内走 ZrevrangebyscoreWithScoresAndLimit 翻过头部回 DB QueryFavoriteList 单页
  - ensureHotHead 用 Zadds 加 Zremrangebyrank 加 Expire 去 Lua 重建
  - 删 QueryUserFavoriteZSetScript 及 query_user_favorite_zset.lua 删分布式锁/轮询/listAllFavorites/内存分页/keepN=5000
  - 接入 capacity/expire 常量 个人流无并发同 key 故不加锁
- ./init.sh 全绿

---

## 已完成

- refactor-001 count canal 消费者重构
  - Consume 拆为解析 路由 去重 落库 派发 五步管道 主文件瘦身
  - 新增 canal_message.go 消息值对象 change_set.go 类型化 key 累积器 effects.go 副作用派发
  - 落库分支收口 consumer.applyUpdate operator 延迟双删裸 go func 改 threading.GoSafe
  - strategy 按判活谓词加目标映射重抽象 presence 与 reset 两族 拆子包 删 followCountTableStrategy 等重复
  - 新增服务级 enum.RecordStatus 复用 pkg/enum.IsDeleted 替换 status 魔法数
  - 删死代码 extractCountUpdates getInt64Value 统一 strategy.ParseInt64
  - 补单测 presence reset registry change_set consumer record_status

## 下一步

1. 工作区未提交 待提交 refactor-003 与 refactor-002 两条（各对应一个提交）
2. 之后从 feature_list.json 选 feat-014~018 中一条新功能开工

## 遗留风险（refactor-003）

- follow 冷备份路径 coldBackfill 查 DB 拿 rows 后只取 ids 再经 L2 resolveDetails 二次回表 仅缓存未命中的冷路径发生 可接受且顺带回填 L2
- 内容本仓无编辑/改可见性写路径 仅 delete 需失效 已接 Invalidate 若后续新增内容编辑须补 Invalidate
- visibility 变更目前不存在 若将来支持 改私密后 L2 仍返回旧 PUBLIC 直到 TTL 收敛 须在该写路径补 Invalidate

---

## 决策记录

- 策略抽象线由 deltaByOpFn 改为判活谓词加目标映射两轴 like/favorite/comment/follow 共用 presenceCounterStrategy
- 子包按实现而非按表拆 presence 一族 reset 一族 空导入触发 init 注册
- applyUpdate 留在 consumer 而非下沉 operator 避免 logic 层反向依赖 mq 层

## 遗留风险

- 坏 JSON 消息 Consume 返回 err 会无限重试 暂无死信处理 后续可加上限或丢弃
