---@diagnostic disable: undefined-global
-- 热榜脏集合冻结脚本 原子执行
-- 把活跃脏集合分片原子地搬到处理中桶 处理期间新互动安全堆进活跃桶
-- KEYS[1] 活跃脏集合 key
-- KEYS[2] 处理中 冻结 脏集合 key
-- 返回 冻结桶成员数 0 表示本轮无脏数据
--
-- 语义
--   1 若活跃桶不存在 直接返回处理中桶现有大小 兜底上轮残留
--   2 若处理中桶已存在 上轮异常未清 先把活跃桶并入处理中桶 再删活跃桶 避免覆盖丢数据
--   3 否则 RENAME 活跃桶到处理中桶 最快路径

local activeKey = KEYS[1]
local procKey = KEYS[2]

if redis.call('EXISTS', activeKey) == 0 then
    return redis.call('SCARD', procKey)
end

if redis.call('EXISTS', procKey) == 1 then
    -- 上一轮 proc 残留 并入而非覆盖 保证不丢
    local members = redis.call('SMEMBERS', activeKey)
    if members ~= nil and #members > 0 then
        redis.call('SADD', procKey, unpack(members))
    end
    redis.call('DEL', activeKey)
else
    redis.call('RENAME', activeKey, procKey)
end

return redis.call('SCARD', procKey)
