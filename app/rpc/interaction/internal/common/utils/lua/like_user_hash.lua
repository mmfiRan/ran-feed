---@diagnostic disable: undefined-global
-- 用户维度点赞 HASH 写入脚本（容量 + mincid + TTL，单次原子）
-- KEYS[1] = userLikeKey (like:user:{user_id})
-- ARGV[1] = content_id
-- ARGV[2] = capacity      热区容量上限
-- ARGV[3] = expire_seconds TTL（0=不过期）
--
-- 返回 {changed, trusted}
--   changed: 1=本次由"未点赞"变为"已点赞"；0=已是点赞态（重复点赞）
--   trusted: 1=缓存完整(_full=1)且该 cid 落在热区，changed 可信；
--            0=缓存残缺或冷数据，changed 不可信，调用方应交由下游兜底
--
-- 注意：本脚本只维护"已存在的完整热区"或"残缺态"，不负责从 DB 全量重建
-- 重建（写入业务字段并置 _full=1）由读路径完成

local key = KEYS[1]
local cid = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2]) or 0
local expire = tonumber(ARGV[3]) or 0

if not cid then
    return {0, 0}
end

local META = {_mincid = true, _full = true}

-- 重算热区最小 cid（仅淘汰时调用，遍历业务字段）
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

local function touchTTL()
    if expire > 0 then redis.call('EXPIRE', key, expire) end
end

local full = redis.call('HGET', key, '_full') == '1'
local minCid = tonumber(redis.call('HGET', key, '_mincid'))
local hotSize = redis.call('HLEN', key)
if minCid ~= nil then hotSize = hotSize - 1 end
if full then hotSize = hotSize - 1 end
if hotSize < 0 then hotSize = 0 end

-- 冷数据：热区已满且 cid 比热区最小值还小 —— 不入缓存，交下游兜底
if minCid ~= nil and hotSize >= capacity and cid < minCid then
    touchTTL()
    return {0, 0}
end

local added = redis.call('HSETNX', key, ARGV[1], '1')
touchTTL()

-- trusted 仅在"缓存完整 且 cid 在热区(>= mincid)"时成立
local trusted = full and (minCid == nil or cid >= minCid)

if added == 0 then
    return {0, trusted and 1 or 0}
end

-- 首次写入：维护容量与 mincid
if minCid ~= nil and hotSize >= capacity and cid >= minCid then
    -- 热区已满，淘汰当前最小 cid 后重算
    redis.call('HDEL', key, tostring(minCid))
    minCid = recalcMinCid()
end

if minCid == nil or cid < minCid then
    redis.call('HSET', key, '_mincid', ARGV[1])
end

return {1, trusted and 1 or 0}