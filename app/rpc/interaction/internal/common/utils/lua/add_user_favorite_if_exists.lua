---@diagnostic disable: undefined-global
-- KEYS[1] = user favorite zset key
-- ARGV[1] = score
-- ARGV[2] = member
-- ARGV[3] = keep_latest_n
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

local keepN = tonumber(ARGV[3])
if keepN ~= nil and keepN > 0 then
    local card = redis.call('ZCARD', key)
    if card ~= nil and card > keepN then
        redis.call('ZREMRANGEBYRANK', key, 0, card - keepN - 1)
    end
end

return 1
