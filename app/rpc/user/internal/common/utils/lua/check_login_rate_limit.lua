---@diagnostic disable: undefined-global
-- 检查登录失败滑动窗口是否已达上限
-- KEYS[1]=zsetKey
-- ARGV[1]=now (秒)
-- ARGV[2]=window (秒)
-- ARGV[3]=maxAttempts
-- 返回: 0 表示允许；>0 表示拒绝并给出建议 retryAfter（秒，向上取整）

local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local maxAttempts = tonumber(ARGV[3])

redis.call("ZREMRANGEBYSCORE", KEYS[1], 0, now - window)
local count = redis.call("ZCARD", KEYS[1])
if count < maxAttempts then
  return 0
end

local oldest = redis.call("ZRANGE", KEYS[1], 0, 0, "WITHSCORES")
local retryAfter = 1
if #oldest >= 2 then
  local oldestScore = tonumber(oldest[2])
  retryAfter = oldestScore + window - now
  if retryAfter < 1 then retryAfter = 1 end
end
return retryAfter