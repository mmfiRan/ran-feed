---@diagnostic disable: undefined-global
-- KEYS[1]=点赞用户集合Key
-- KEYS[2]=点赞计数Key
-- ARGV[1]=用户ID（可选，为空表示未登录）
-- ARGV[2]=过期时间(秒)
-- 返回: {exists, count, isLiked}
--   exists: 1=缓存存在, 0=缓存不存在
--   count: 点赞计数（如果exists=0则无意义）
--   isLiked: 1=已点赞, 0=未点赞（如果exists=0或无用户ID则为0）

local expireTime = tonumber(ARGV[2]) or 432000
local userID = ARGV[1]

-- 检查点赞计数key是否存在
local countExists = redis.call('EXISTS', KEYS[2])
if countExists == 0 then
    -- 缓存不存在
    return {0, 0, 0}
end

-- 获取点赞计数
local count = redis.call('GET', KEYS[2]) or '0'

-- 刷新过期时间
redis.call('EXPIRE', KEYS[2], expireTime)
redis.call('EXPIRE', KEYS[1], expireTime)

-- 如果没有用户ID，直接返回
if userID == '' then
    return {1, tonumber(count), 0}
end

-- 检查用户是否点赞
local isLiked = redis.call('SISMEMBER', KEYS[1], userID)

return {1, tonumber(count), isLiked}
