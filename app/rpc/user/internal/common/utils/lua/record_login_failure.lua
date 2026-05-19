---@diagnostic disable: undefined-global
-- 记录一次登录失败到滑动窗口 zset
-- KEYS[1]=zsetKey
-- ARGV[1]=now (秒)
-- ARGV[2]=window (秒)
-- ARGV[3]=唯一 member（同一秒多次失败避免覆盖）
-- 返回: 记录后的当前窗口失败次数

local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

redis.call("ZREMRANGEBYSCORE", KEYS[1], 0, now - window)
redis.call("ZADD", KEYS[1], now, ARGV[3])
redis.call("EXPIRE", KEYS[1], window)
return redis.call("ZCARD", KEYS[1])