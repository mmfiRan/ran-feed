---@diagnostic disable: undefined-global
-- 用户维度批量查询是否点赞 HASH 脚本（TTL，原子性）
-- KEYS[1]=userLikeKey (like:user:{user_id})
-- ARGV[1]=expire_seconds
-- ARGV[2...]=content_id 列表
-- 返回: {exists(0/1), mincid(string), liked1(0/1), liked2(0/1), ...}
-- likedN 顺序与 ARGV[2...] 一致

local expireTime = tonumber(ARGV[1]) or 0

local exists = redis.call('EXISTS', KEYS[1])
if exists == 0 then
    return {0, ''}
end

local mincid = redis.call('HGET', KEYS[1], '_mincid') or ''
local result = {1, mincid}

for i = 2, #ARGV do
    local liked = redis.call('HEXISTS', KEYS[1], ARGV[i])
    table.insert(result, liked)
end

if expireTime > 0 then
    redis.call('EXPIRE', KEYS[1], expireTime)
end

return result
