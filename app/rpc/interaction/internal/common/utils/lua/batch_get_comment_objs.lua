---@diagnostic disable: undefined-global
-- 批量读取评论对象HASH（并刷新TTL）
-- KEYS[i]=评论对象key comment:obj:{commentId}
-- ARGV[1]=obj_ttl_seconds
-- ARGV[i+1]=comment_id (与KEYS一一对应)
-- 返回: {id1, content_id1, user_id1, reply_to_user_id1, parent_id1, root_id1, comment1, created_at1, status1, user_name1, user_avatar1, reply_count1, id2, ...}

local objTtl = tonumber(ARGV[1])
local res = {}

for i = 1, #KEYS do
    local id = ARGV[i + 1] or ''
    local objKey = KEYS[i]

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
