---@diagnostic disable: undefined-global
-- Redis评论缓存更新Lua脚本
-- KEYS[1]=评论对象key comment:obj:{commentId}
-- KEYS[2]=内容评论索引key comment:idx:content:{contentId}
-- ARGV[1]=obj_ttl_seconds
-- ARGV[2]=idx_ttl_seconds
-- ARGV[3]=keep_latest_n
-- ARGV[4]=comment_id
-- ARGV[5]=content_id
-- ARGV[6]=user_id
-- ARGV[7]=reply_to_user_id
-- ARGV[8]=parent_id
-- ARGV[9]=root_id
-- ARGV[10]=comment
-- ARGV[11]=created_at
-- ARGV[12]=status
-- ARGV[13]=user_name
-- ARGV[14]=user_avatar
-- ARGV[15]=reply_count
-- 返回: 1

redis.call('HSET', KEYS[1],
    'comment_id', ARGV[4],
    'content_id', ARGV[5],
    'user_id', ARGV[6],
    'reply_to_user_id', ARGV[7],
    'parent_id', ARGV[8],
    'root_id', ARGV[9],
    'comment', ARGV[10],
    'created_at', ARGV[11],
    'status', ARGV[12],
    'user_name', ARGV[13],
    'user_avatar', ARGV[14],
    'reply_count', ARGV[15]
)

redis.call('EXPIRE', KEYS[1], ARGV[1])
redis.call('ZADD', KEYS[2], ARGV[4], ARGV[4])
redis.call('EXPIRE', KEYS[2], ARGV[2])

local card = redis.call('ZCARD', KEYS[2])
local keepN = tonumber(ARGV[3])
if keepN ~= nil and keepN > 0 and card > keepN then
    redis.call('ZREMRANGEBYRANK', KEYS[2], 0, card - keepN - 1)
end

return 1
