---@diagnostic disable: undefined-global
-- Redis关注收件箱回填并返回实际新增数量Lua脚本
-- KEYS[1] = inbox zset key
-- ARGV[1] = keep_latest_n
-- ARGV[2] = cutoff_millis 早于此 score 的成员裁剪 <=0 跳过
-- ARGV[3] = ttl_seconds 整 key 续期 <=0 跳过
-- ARGV[4...] = score1, member1, score2, member2, ...
-- 返回: 实际新增数量

local key = KEYS[1]
local keepN = tonumber(ARGV[1])
local cutoff = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

-- ZADD 返回新增成员数 累加即实际新增 省去末尾 N 次 ZSCORE 回查
local added = 0
for i = 4, #ARGV, 2 do
    local score = ARGV[i]
    local member = ARGV[i + 1]
    if score ~= nil and member ~= nil and member ~= '' then
        added = added + redis.call('ZADD', key, score, member)
    end
end

-- 时间窗口裁剪 删早于 cutoff 的成员
if cutoff ~= nil and cutoff > 0 then
    redis.call('ZREMRANGEBYSCORE', key, '-inf', '(' .. cutoff)
end

-- 条数兜底裁剪
if keepN ~= nil and keepN > 0 then
    local card = redis.call('ZCARD', key)
    if card ~= nil and card > keepN then
        redis.call('ZREMRANGEBYRANK', key, 0, card - keepN - 1)
    end
end

-- 整 key 续期 活跃读写存活 冷用户整 key 过期回收
if ttl ~= nil and ttl > 0 then
    redis.call('EXPIRE', key, ttl)
end

return added
