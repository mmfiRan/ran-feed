---@diagnostic disable: undefined-global
-- KEYS[1] = user publish zset key
-- ARGV[1] = cursor member (content_id string), empty means first page
-- ARGV[2] = page size
-- return: {exists, has_more, next_cursor, id1, id2, ...}

local key = KEYS[1]
local cursor = ARGV[1]
local pageSize = tonumber(ARGV[2])

local exists = redis.call('EXISTS', key)
if exists == 0 then
    return {0, 0, ""}
end

if pageSize == nil then
    return {1, 0, ""}
end

local maxScore = "+inf"
if cursor ~= nil and cursor ~= "" then
    maxScore = cursor
end

local ids = redis.call('ZREVRANGEBYSCORE', key, '(' .. maxScore, '-inf', 'LIMIT', 0, pageSize + 1)

local hasMore = 0
if #ids > pageSize then
    hasMore = 1
end

local nextCursor = ""
if hasMore == 1 then
    nextCursor = ids[pageSize]
end

local res = {1, hasMore, nextCursor}
for i = 1, math.min(#ids, pageSize) do
    res[#res + 1] = ids[i]
end

return res
