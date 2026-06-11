# 会话进度日志

## 当前状态

**最后更新：** 2026-06-11
**当前功能：** refactor-002 收藏链路去 Lua 化（in-progress 待决 content 侧方向）

---

## 进行中（refactor-002 收藏链路去 Lua 化）

- interaction 侧已完成（未提交 工作区）
  - 删 favorite:rel 关系缓存与负缓存 及 BuildFavoriteRelKey 删 add_user_favorite_if_exists.lua 及 Go 变量
  - QueryFavoriteInfo 去掉关系缓存与分布式锁重建 改直接回源 DB favoriteRepo.IsFavorited
  - Favorite/RemoveFavorite 改纯 cache-aside 仅删用户收藏 feed 头部缓存
  - 顺带精简 batch_query_is_liked 注释 distlock.go 注释
- content 侧本会话止血
  - 还原 query_user_favorite_zset.lua 及 QueryUserFavoriteZSetScript 修复编译 ./init.sh 通过
  - 删无人引用的 add_user_favorite_if_exists.lua 及变量
  - 新增常量 RedisUserFavoriteFeedCapacity=300 RedisUserFavoriteFeedExpireSeconds=24h 尚未接入
- 待用户决定 content user_favorite_feed_logic 是否去 Lua 化接入新常量

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

1. 确定 content 侧收藏 feed 去 Lua 化方向 完成 refactor-002 或回退
2. refactor-002 收尾后 从 feature_list.json 选 feat-014~018 中一条新功能开工

---

## 决策记录

- 策略抽象线由 deltaByOpFn 改为判活谓词加目标映射两轴 like/favorite/comment/follow 共用 presenceCounterStrategy
- 子包按实现而非按表拆 presence 一族 reset 一族 空导入触发 init 注册
- applyUpdate 留在 consumer 而非下沉 operator 避免 logic 层反向依赖 mq 层

## 遗留风险

- 坏 JSON 消息 Consume 返回 err 会无限重试 暂无死信处理 后续可加上限或丢弃
