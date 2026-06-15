# 会话进度日志

## 当前状态

**最后更新：** 2026-06-15
**当前功能：** refactor-003 feed 二级缓存内容详情统一（done）+ refactor-002 收藏链路去 Lua 化（done）

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
