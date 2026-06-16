---@diagnostic disable: undefined-global
-- KEYS[1] = inbox zset key
-- ARGV[1] = cursor 上一页末位 score(published_at 毫秒) 空表示首页
-- ARGV[2] = page size
-- ARGV[3] = cutoff_millis 窗口下界 仅返回 score >= cutoff 的成员 <=0 不限
-- ARGV[4] = ttl_seconds 命中即续期整 key <=0 跳过
-- return: {exists, has_more, next_cursor, member1, score1, member2, score2, ...}

local key = KEYS[1]
local cursor = ARGV[1]
local pageSize = tonumber(ARGV[2])
local cutoff = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local exists = redis.call('EXISTS', key)
if exists == 0 then
    return {0, 0, ""}
end

if pageSize == nil then
    return {1, 0, ""}
end

local maxScore = "+inf"
if cursor ~= nil and cursor ~= "" and cursor ~= "0" then
    maxScore = "(" .. cursor
end

local minScore = "-inf"
if cutoff ~= nil and cutoff > 0 then
    minScore = cutoff
end

local items = redis.call('ZREVRANGEBYSCORE', key, maxScore, minScore, 'WITHSCORES', 'LIMIT', 0, pageSize + 1)
local pairCount = math.floor(#items / 2)

local hasMore = 0
if pairCount > pageSize then
    hasMore = 1
end

local nextCursor = ""
if hasMore == 1 then
    -- 第 pageSize 个成员的 score 作下一页游标
    nextCursor = items[pageSize * 2]
end

if ttl ~= nil and ttl > 0 then
    redis.call('EXPIRE', key, ttl)
end

local res = {1, hasMore, nextCursor}
local limit = math.min(pairCount, pageSize)
for i = 1, limit do
    res[#res + 1] = items[i * 2 - 1]
    res[#res + 1] = items[i * 2]
end

return res
