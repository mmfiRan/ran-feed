---@diagnostic disable: undefined-global
-- Redis关注收件箱回填/更新Lua脚本
-- KEYS[1] = inbox zset key
-- ARGV[1] = keep_latest_n
-- ARGV[2...] = score1, member1, score2, member2, ...
-- 返回: 1

local key = KEYS[1]
local keepN = tonumber(ARGV[1])

for i = 2, #ARGV, 2 do
    local score = ARGV[i]
    local member = ARGV[i + 1]
    if score ~= nil and member ~= nil and member ~= '' then
        redis.call('ZADD', key, score, member)
    end
end

if keepN ~= nil and keepN > 0 then
    local card = redis.call('ZCARD', key)
    if card ~= nil and card > keepN then
        redis.call('ZREMRANGEBYRANK', key, 0, card - keepN - 1)
    end
end

return 1
