---@diagnostic disable: undefined-global
-- 用户维度取消点赞 HASH 脚本（mincid + TTL，单次原子）
-- KEYS[1] = userLikeKey (like:user:{user_id})
-- ARGV[1] = content_id
-- ARGV[2] = expire_seconds TTL（0=不过期）
--
-- 返回 {changed, trusted}
--   changed: 1=本次确实删除了点赞态；0=缓存中本就没有该 cid
--   trusted: 1=缓存完整(_full=1)且该 cid 落在热区，changed 可信；
--            0=缓存残缺或冷数据，changed 不可信，调用方应交由下游兜底

local key = KEYS[1]
local cid = tonumber(ARGV[1])
local expire = tonumber(ARGV[2]) or 0

if not cid then
    return {0, 0}
end

local META = {_mincid = true, _full = true}

local function recalcMinCid()
    local fields = redis.call('HKEYS', key)
    local min = nil
    for i = 1, #fields do
        local f = fields[i]
        if not META[f] then
            local n = tonumber(f)
            if n and (not min or n < min) then min = n end
        end
    end
    return min
end

local full = redis.call('HGET', key, '_full') == '1'
local minCid = tonumber(redis.call('HGET', key, '_mincid'))
local trusted = full and (minCid == nil or cid >= minCid)

local removed = redis.call('HDEL', key, ARGV[1])
if expire > 0 then redis.call('EXPIRE', key, expire) end

if removed == 0 then
    return {0, trusted and 1 or 0}
end

-- 删除的恰好是热区最小 cid，重算 mincid
if minCid ~= nil and cid == minCid then
    local newMin = recalcMinCid()
    if newMin ~= nil then
        redis.call('HSET', key, '_mincid', tostring(newMin))
    else
        redis.call('HDEL', key, '_mincid')
    end
end

return {1, trusted and 1 or 0}