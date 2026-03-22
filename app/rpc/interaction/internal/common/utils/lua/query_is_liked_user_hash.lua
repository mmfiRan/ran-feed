---@diagnostic disable: undefined-global
-- 用户维度查询是否点赞 HASH 脚本（TTL，原子性）
-- KEYS[1]=userLikeKey (like:user:{user_id})
-- ARGV[1]=content_id
-- ARGV[2]=expire_seconds
-- 返回: {exists(0/1), isLiked(0/1), mincid(string)}

local expireTime = tonumber(ARGV[2]) or 0

local exists = redis.call('EXISTS', KEYS[1])
if exists == 0 then
    return {0, 0, ''}
end

local liked = redis.call('HEXISTS', KEYS[1], ARGV[1])
local mincid = redis.call('HGET', KEYS[1], '_mincid') or ''

if expireTime > 0 then
    redis.call('EXPIRE', KEYS[1], expireTime)
end

return {1, liked, mincid}
