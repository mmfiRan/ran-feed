---@diagnostic disable: undefined-global
-- 热榜ZSET重建/回填Lua脚本
-- KEYS[1] = zset key
-- ARGV = score1, member1, score2, member2, ...
-- 返回: 1

local key = KEYS[1]

if ARGV == nil or #ARGV == 0 then
    return 1
end

for i = 1, #ARGV, 2 do
    local score = ARGV[i]
    local member = ARGV[i + 1]
    if score ~= nil and member ~= nil and member ~= '' then
        redis.call('ZADD', key, score, member)
    end
end

return 1
