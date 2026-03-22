---@diagnostic disable: undefined-global
-- 用户维度取消点赞 HASH 脚本（TTL，原子性）
-- KEYS[1]=userLikeKey (like:user:{user_id})
-- ARGV[1]=content_id
-- ARGV[2]=expire_seconds
-- 返回: {changed(0/1), existed(0/1)}

local expireTime = tonumber(ARGV[2]) or 0

local removed = redis.call('HDEL', KEYS[1], ARGV[1])
if expireTime > 0 then
    redis.call('EXPIRE', KEYS[1], expireTime)
end

if removed == 0 then
    return {0, 0}
end

-- 若删除的是当前 mincid，则需要重算新的 mincid（扫描所有业务 field）
local minCidStr = redis.call('HGET', KEYS[1], '_mincid')
if minCidStr ~= false and minCidStr == ARGV[1] then
    local fields = redis.call('HKEYS', KEYS[1])
    local newMin = nil
    for i = 1, #fields do
        local f = fields[i]
        if string.sub(f, 1, 1) ~= '_' then
            local n = tonumber(f)
            if n ~= nil then
                if newMin == nil or n < newMin then
                    newMin = n
                end
            end
        end
    end
    if newMin ~= nil then
        redis.call('HSET', KEYS[1], '_mincid', tostring(newMin))
    else
        redis.call('HDEL', KEYS[1], '_mincid')
    end
end

return {1, 1}
