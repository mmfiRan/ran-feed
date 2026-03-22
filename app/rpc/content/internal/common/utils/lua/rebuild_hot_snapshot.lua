---@diagnostic disable: undefined-global
-- 热榜快照重建脚本（原子执行）
-- KEYS[1] = 热榜 zset key
-- KEYS[2] = 快照 zset key
-- KEYS[3] = 最新快照 id key
-- ARGV[1] = topN
-- ARGV[2] = snapshotId
-- ARGV[3] = ttlSeconds
-- 返回：{count}

local zsetKey = KEYS[1]
local snapshotKey = KEYS[2]
local latestKey = KEYS[3]
local topN = tonumber(ARGV[1])
local snapshotId = ARGV[2]
local ttlSeconds = tonumber(ARGV[3])

if topN == nil or topN <= 0 then
    return {0}
end

local raw = redis.call('ZREVRANGE', zsetKey, 0, topN - 1, 'WITHSCORES')
if raw == nil or #raw == 0 then
    return {0}
end

redis.call('DEL', snapshotKey)
local count = 0
for i = 1, #raw, 2 do
    local member = raw[i]
    local score = raw[i + 1]
    if member ~= nil and member ~= '' and score ~= nil then
        redis.call('ZADD', snapshotKey, score, member)
        count = count + 1
    end
end

if ttlSeconds ~= nil and ttlSeconds > 0 then
    redis.call('EXPIRE', snapshotKey, ttlSeconds)
end

if snapshotId ~= nil and snapshotId ~= '' then
    redis.call('SET', latestKey, snapshotId)
end

return {count}
