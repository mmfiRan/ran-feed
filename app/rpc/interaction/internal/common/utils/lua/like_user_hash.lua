---@diagnostic disable: undefined-global
-- 用户维度点赞 HASH 写入脚本（容量+mincid+TTL，原子性）
-- KEYS[1]=userLikeKey (like:user:{user_id})
-- ARGV[1]=content_id
-- ARGV[2]=capacity (e.g. 10000)
-- ARGV[3]=expire_seconds
-- 返回: {changed(0/1), cached(0/1)}
--   changed: 1=本次确实从未点赞->已点赞；0=重复点赞
--   cached: 1=写入了缓存；0=未写入缓存（容量满且 cid < mincid）

local cid = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2]) or 10000
local expireTime = tonumber(ARGV[3]) or 0

if not cid then
    return {0, 0}
end

-- 读取元信息
local minCidStr = redis.call('HGET', KEYS[1], '_mincid')
local minCid = tonumber(minCidStr)

-- 用 HLEN 做容量判定
local hlen = redis.call('HLEN', KEYS[1])
local metaCount = 0
if redis.call('HEXISTS', KEYS[1], '_mincid') == 1 then metaCount = metaCount + 1 end
if redis.call('HEXISTS', KEYS[1], '_expire_at') == 1 then metaCount = metaCount + 1 end
if redis.call('HEXISTS', KEYS[1], '_ver') == 1 then metaCount = metaCount + 1 end

local realSize = hlen - metaCount
if realSize < 0 then realSize = 0 end

-- 当容量已满且当前 cid 比 mincid 还小，说明是冷数据，不入缓存
if minCid ~= nil and realSize >= capacity and cid < minCid then
    if expireTime > 0 then
        redis.call('EXPIRE', KEYS[1], expireTime)
    end
    return {1, 0}
end

-- 正常写入缓存（HSETNX）
local added = redis.call('HSETNX', KEYS[1], ARGV[1], '1')
if expireTime > 0 then
    redis.call('EXPIRE', KEYS[1], expireTime)
end

if added == 0 then
    return {0, 1}
end

-- added==1，首次点赞

-- 若容量已满，且本次写入的 cid 不小于 mincid，则淘汰当前最小 cid
-- 注意：这里的容量基于写入前 realSize 判定，写入后会 +1，因此用 "realSize >= capacity" 判断是否需要淘汰
if minCid ~= nil and realSize >= capacity and cid >= minCid then
    redis.call('HDEL', KEYS[1], tostring(minCid))

    -- 重新计算新的 mincid：遍历所有 field，挑出最小的数字 field（忽略 '_' 前缀元字段）
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
        minCid = newMin
    else
        redis.call('HDEL', KEYS[1], '_mincid')
        minCid = nil
    end
end

-- 初始化/更新 mincid
if minCid == nil then
    redis.call('HSET', KEYS[1], '_mincid', ARGV[1])
else
    if cid < minCid then
        redis.call('HSET', KEYS[1], '_mincid', ARGV[1])
    end
end

-- 预留字段
if redis.call('HEXISTS', KEYS[1], '_expire_at') == 0 then
    redis.call('HSET', KEYS[1], '_expire_at', '0')
end
if redis.call('HEXISTS', KEYS[1], '_ver') == 0 then
    redis.call('HSET', KEYS[1], '_ver', '1')
end

return {1, 1}
