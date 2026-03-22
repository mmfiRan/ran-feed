---@diagnostic disable: undefined-global
-- KEYS[1] = 优先快照 zset key（可为空）
-- KEYS[2] = 最新快照 id key
-- KEYS[3] = 快照 key 前缀（例如 feed:hot:global:snap）
-- KEYS[4] = 全站热榜 zset key
-- ARGV[1] = cursor 成员（content_id 字符串），空表示首页
-- ARGV[2] = page size
-- ARGV[3] = 优先快照 id（可为空）
-- return: {exists, has_more, next_cursor, resolved_snapshot_id, id1, id2, ...}

local preferredKey = KEYS[1]
local latestKey = KEYS[2]
local snapshotPrefix = KEYS[3]
local globalKey = KEYS[4]
local cursor = ARGV[1]
local pageSize = tonumber(ARGV[2])
local preferredSnapshotId = ARGV[3]

local key = ""
local resolvedSnapshotId = ""

if preferredKey ~= nil and preferredKey ~= "" then
    local existsPreferred = redis.call('EXISTS', preferredKey)
    if existsPreferred == 1 then
        key = preferredKey
        resolvedSnapshotId = preferredSnapshotId or ""
    end
end

if key == "" then
    local latestId = redis.call('GET', latestKey)
    if latestId ~= nil and latestId ~= "" then
        local latestSnapshotKey = snapshotPrefix .. ":" .. latestId
        local existsLatest = redis.call('EXISTS', latestSnapshotKey)
        if existsLatest == 1 then
            key = latestSnapshotKey
            resolvedSnapshotId = latestId
        end
    end
end

if key == "" then
    key = globalKey
    resolvedSnapshotId = ""
end

local exists = redis.call('EXISTS', key)
if exists == 0 then
    return {0, 0, "", resolvedSnapshotId}
end

if pageSize == nil then
    return {1, 0, "", resolvedSnapshotId}
end

local cursorScore = nil
local cursorId = nil
if cursor ~= nil and cursor ~= "" then
    cursorScore = redis.call('ZSCORE', key, cursor)
    cursorId = tonumber(cursor)
end

local maxScore = "+inf"
if cursorScore ~= nil then
    maxScore = cursorScore
end

local overscan = pageSize + 32

local raw = redis.call('ZREVRANGEBYSCORE', key, '(' .. maxScore, '-inf', 'WITHSCORES', 'LIMIT', 0, overscan)
local ids = {}

for i = 1, #raw, 2 do
    local member = raw[i]
    local score = raw[i + 1]

    local memberId = tonumber(member)
    if cursorScore ~= nil and score == cursorScore and memberId ~= nil and cursorId ~= nil and memberId >= cursorId then
    else
        ids[#ids + 1] = member
        if #ids >= (pageSize + 1) then
            break
        end
    end
end

local hasMore = 0
if #ids > pageSize then
    hasMore = 1
end

local nextCursor = ""
if hasMore == 1 then
    nextCursor = ids[pageSize]
end

local res = {1, hasMore, nextCursor, resolvedSnapshotId}
for i = 1, math.min(#ids, pageSize) do
    res[#res + 1] = ids[i]
end

return res
