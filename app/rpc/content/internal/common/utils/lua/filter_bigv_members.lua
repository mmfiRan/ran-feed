---@diagnostic disable: undefined-global
-- 批量判定候选 id 是否在全局大 V 集合中
-- 成本随候选数增长 而非随集合体量增长 支撑棘轮语义下集合单调增长
-- KEYS[1] = 全局大 V 集合 key
-- ARGV[1..] = 候选 user_id
-- 返回: 命中的 user_id 数组

local key = KEYS[1]
local hit = {}
for i = 1, #ARGV do
    local id = ARGV[i]
    if id ~= nil and id ~= '' and redis.call('SISMEMBER', key, id) == 1 then
        hit[#hit + 1] = id
    end
end
return hit
