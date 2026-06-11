# 缓存规范（cache-aside）

读路径加缓存统一走旁路缓存，参考 `app/rpc/user/internal/common/utils/usercache/`：

- **正负值都缓存**：命中正值反序列化返回；DB 也 miss 时写**负值哨兵**（如 `"-"`），下次直接返回不再回源，防穿透
- **TTL 叠加 jitter 抗雪崩**：正负 TTL 都加 `[0, JitterMax]` 随机抖动，避免同一时刻集中过期
- **缓存只降级不阻断**：任何 Redis 错误只记日志并回源 DB，不要把缓存错误冒泡给调用方
- **批量用一次 RTT**：批查用 `MgetCtx` / `PipelinedCtx`，回写也合并，不要循环单查
- **缓存只存必要字段子集**：敏感字段（密码 盐 邮箱）不进缓存
- **更新/删除要失效缓存**：写路径调用 `Invalidate` 删除对应 key
- **防击穿分层选型**（`pkg/cache`）：per-user 类 key 用进程内单飞 `cache.Group` + `cache.Do`；
  全集群共享的热点 key（全局热榜 全网热门详情 全局配置）用分布式锁 `cache.DistLocker` + `DoWithLock`，锁 key 统一 `cache.BuildLockKey`