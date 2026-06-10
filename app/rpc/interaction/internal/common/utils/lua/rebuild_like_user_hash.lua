---@diagnostic disable: undefined-global
-- 用户维度点赞 HASH 重建脚本（原子重建，单次原子）
-- 用法 读路径发现 _full=0 或 key 不存在 时 由调用方从 DB 拉 topN 调本脚本一次性写入完整热区
--
-- KEYS[1] = userLikeKey (like:user:{user_id})
-- ARGV[1] = expire_seconds TTL（0=不过期）
-- ARGV[2] = mincid 字符串 ""=无业务字段（用户无任何点赞 写空 _full=1 hash 避免反复重建）
-- ARGV[3..] = cid 列表（每个 cid 作为 hash field 值统一为 "1"）
--
-- 返回 1=重建成功
--
-- 注意 先 DEL 再写 避免残留冷字段或旧 _mincid 污染热区语义

redis.call('DEL', KEYS[1])

local n = #ARGV
if n >= 3 then
    for i = 3, n do
        redis.call('HSET', KEYS[1], ARGV[i], '1')
    end
end

if ARGV[2] ~= '' then
    redis.call('HSET', KEYS[1], '_mincid', ARGV[2])
end

redis.call('HSET', KEYS[1], '_full', '1')

local expire = tonumber(ARGV[1]) or 0
if expire > 0 then
    redis.call('EXPIRE', KEYS[1], expire)
end

return 1
