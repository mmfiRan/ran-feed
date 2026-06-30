# 会话进度日志

## 当前状态

**最后更新：** 2026-06-30
**当前功能：** feat-003 SearchService 查询逻辑（done）— 搜索功能 P4

---

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
