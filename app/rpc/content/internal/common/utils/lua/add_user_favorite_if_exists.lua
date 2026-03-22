---@diagnostic disable: undefined-global
-- KEYS[1] = user favorite zset key
-- ARGV[1] = score
-- ARGV[2] = member
-- return: 1 if updated, 0 if key not exists

local key = KEYS[1]
local exists = redis.call('EXISTS', key)
if exists == 0 then
    return 0
end

local score = ARGV[1]
local member = ARGV[2]
if score ~= nil and member ~= nil and member ~= '' then
    redis.call('ZADD', key, score, member)
end

return 1
