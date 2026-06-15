# 会话交接（Session Handoff）

> 每个会话结束时更新本文件，下一会话启动时阅读。只描述**最新一次交接的当前面**。

最后更新：2026-06-15

---

## 当前目标

- **目标**：refactor-003 feed 二级缓存内容详情统一 + refactor-002 收藏链路去 Lua 化（content 侧收口）
- **当前状态**：两条均 done ./init.sh 通过 工作区未提交 待提交（各对应一个提交）
- **分支 / 提交**：feature/feat 领先 origin 3 个提交（refactor-003 与 refactor-002 content 侧尚未提交）

## 本会话已完成（refactor-003）

- [x] 设计 L1 排序 ZSET 各 feed 自管 新增 L2 按 content_id 缓存内容本征详情 与观察者无关
- [x] 模型 do.ContentDetailDO 六字段加 visibility；新增 contentcache 包 BatchGet/Invalidate（String+JSON+MGET 负哨兵 jitter 只降级 不做击穿 与 usercache 同构）+ 单测
- [x] ContentCacheConfig 走 json default 不写进 content.yaml
- [x] 共享 contentDetailResolver（loadDetails 回源 / resolveDetails 加 publicOnly 过滤 / loadAuthorsAndLikes / assembleItems）
- [x] 四条 feed 接入：user_favorite user_publish recommend follow 各删一份重复拼装；recommend/follow 读时过滤 PUBLIC；follow 出 FollowFeedItem 用 resolveDetails+buildFollowItems
- [x] delete_content 接 contentcache.Invalidate
- [x] ./init.sh build vet test 全绿

## 验证证据

| 检查 | 命令 | 结果 | 备注 |
|------|------|------|------|
| contentcache 单测 | `go test ./app/rpc/content/internal/common/utils/contentcache/...` | 通过 ✓ | 命中/miss/负哨兵/去重/降级/Invalidate |
| 编译 静态分析 测试 | `./init.sh` | 通过 ✓ | build vet test 全绿 |

## 改动文件（工作区未提交 均在 content 服务）

- internal/do/content_do.go 新增 ContentDetailDO
- internal/common/consts/redis/redis_consts.go 加 content:detail 前缀/负哨兵/BuildContentDetailKey
- internal/config/config.go 加 ContentCacheConfig
- internal/common/utils/contentcache/cache.go + cache_test.go（新）
- internal/logic/feedservice/content_detail_resolver.go（新 共享件）
- internal/logic/feedservice/{user_favorite,user_publish,recommend,follow}_feed_logic.go 接入并删重复拼装
- internal/logic/contentservice/delete_content_logic.go 接 Invalidate

## 决策记录

- 二级缓存用 String+JSON+MGET 不用 Hash：访问模式是整取整存 无字段级读写 Hash 优势空转且批量读退化 与 usercache 同构
- 不做击穿防护：L2 重建廉价（主键 IN 批查）jitter 错峰足够 分布式锁留给重的 L1 ZSET 重建
- author 名头像不进 L2：走 user 服务已有 usercache 避免作者改名要反向失效海量内容详情
- visibility 进 L2：recommend/follow 读时按 PUBLIC 过滤 publish/favorite 不过滤（保持原行为）

## refactor-002 content 侧改动文件（本会话新增 未提交）

- internal/logic/feedservice/user_favorite_feed_logic.go 改热头部旁路缓存 + 去 Lua（loadPageIDs/ensureHotHead/pageFromDB）
- internal/common/utils/lua/redis_lua.go 删 QueryUserFavoriteZSetScript 变量
- internal/common/utils/lua/query_user_favorite_zset.lua 删除文件

## 风险 / 阻塞

- follow 冷备份 coldBackfill 查 DB 后只取 ids 再经 L2 二次回表 仅冷路径 可接受且回填 L2
- 本仓无内容编辑/改可见性写路径 仅 delete 接了 Invalidate 若后续新增编辑须补 Invalidate
- 收藏空集用户每次都会 EXISTS miss 触发一次 QueryFavoriteList 重建（不建 key）频率低可接受 必要时加空 sentinel
- 收藏热头部边界页：翻到第 ~300 条附近时整页改由 DB 出（不混拼不出短页）符合预期

## 下一会话启动步骤

1. 阅读 `CLAUDE.md`
2. `./init.sh` 验证环境
3. 阅读 `feature_list.json` + `progress.md` + 本文件

## 推荐下一步

- 提交两条（git-commit skill 一个提交对应一个条目）：refactor-003 feed 二级缓存内容详情统一；refactor-002 收藏链路去 Lua 化 content 侧
- 之后从 feature_list.json 选 feat-014~018 一条新功能开工