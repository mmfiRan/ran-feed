---@diagnostic disable: undefined-global
-- 更新评论对象HASH（仅对象，不更新任何索引）
-- KEYS[1]=评论对象key comment:obj:{commentId}
-- ARGV[1]=obj_ttl_seconds
-- ARGV[2]=comment_id
-- ARGV[3]=content_id
-- ARGV[4]=user_id
-- ARGV[5]=reply_to_user_id
-- ARGV[6]=parent_id
-- ARGV[7]=root_id
-- ARGV[8]=comment
-- ARGV[9]=created_at
-- ARGV[10]=status
-- ARGV[11]=user_name
-- ARGV[12]=user_avatar
-- ARGV[13]=reply_count
-- 返回: 1

redis.call('HSET', KEYS[1],
    'comment_id', ARGV[2],
    'content_id', ARGV[3],
    'user_id', ARGV[4],
    'reply_to_user_id', ARGV[5],
    'parent_id', ARGV[6],
    'root_id', ARGV[7],
    'comment', ARGV[8],
    'created_at', ARGV[9],
    'status', ARGV[10],
    'user_name', ARGV[11],
    'user_avatar', ARGV[12],
    'reply_count', ARGV[13]
)
redis.call('EXPIRE', KEYS[1], ARGV[1])
return 1
