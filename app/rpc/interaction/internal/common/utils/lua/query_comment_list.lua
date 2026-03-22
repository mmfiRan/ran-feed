---@diagnostic disable: undefined-global
-- Redis评论列表查询Lua脚本（索引ZSET + 对象缓存）
-- KEYS[1]=内容评论索引key comment:idx:content:{contentId}
-- ARGV[1]=cursor (0表示第一页)
-- ARGV[2]=page_size
-- ARGV[3]=idx_ttl_seconds
-- ARGV[4]=obj_ttl_seconds
-- 返回:
--   {exists, next_cursor, has_more, id1, content_id1, user_id1, reply_to_user_id1, parent_id1, root_id1, comment1, created_at1, status1, user_name1, user_avatar1, reply_count1, id2, ...}
--   exists: 1=索引存在, 0=索引不存在

local cursor = tonumber(ARGV[1]) or 0
local pageSize = tonumber(ARGV[2]) or 20
local idxTtl = tonumber(ARGV[3])
local objTtl = tonumber(ARGV[4])

if (not pageSize) or pageSize <= 0 then
    pageSize = 20
end

local idxExists = redis.call('EXISTS', KEYS[1])
if idxExists == 0 then
    return {0, 0, 0}
end

if idxTtl and idxTtl > 0 then
    redis.call('EXPIRE', KEYS[1], idxTtl)
end

local maxScore = '+inf'
if cursor and cursor > 0 then
    maxScore = tostring(cursor - 1)
end

local fetchSize = pageSize + 1
local ids = redis.call('ZREVRANGEBYSCORE', KEYS[1], maxScore, '-inf', 'LIMIT', 0, fetchSize)
local nextCursor = 0
local hasMore = 0
if ids and #ids > pageSize then
    hasMore = 1
    ids[#ids] = nil
end
if ids and #ids > 0 then
    nextCursor = tonumber(ids[#ids]) or 0
end

local res = {1, nextCursor, hasMore}
for i = 1, #ids do
    local id = ids[i]
    local objKey = 'comment:obj:' .. id
    local vals = redis.call('HMGET', objKey,
        'content_id',
        'user_id',
        'reply_to_user_id',
        'parent_id',
        'root_id',
        'comment',
        'created_at',
        'status',
        'user_name',
        'user_avatar',
        'reply_count'
    )
    if objTtl and objTtl > 0 then
        redis.call('EXPIRE', objKey, objTtl)
    end

    res[#res+1] = id
    if vals == false or vals == nil then
        for _ = 1, 11 do
            res[#res+1] = ''
        end
    else
        for j = 1, 11 do
            res[#res+1] = vals[j] or ''
        end
    end
end

return res
