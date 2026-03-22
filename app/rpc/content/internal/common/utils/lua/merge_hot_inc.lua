---@diagnostic disable: undefined-global
-- 热榜增量合并脚本（原子执行）
-- KEYS[1] = 增量 hash key
-- KEYS[2] = 热榜 zset key
-- ARGV[1] = score 精度保留小数位（整数）
-- ARGV[2...] 可选：member, delta, member, delta ...（显式增量）
-- 返回：{merged_count}

local incKey = KEYS[1]
local zsetKey = KEYS[2]
local precision = tonumber(ARGV[1]) or 0

-- 两种输入模式：
-- 1) 新模式（推荐）：Go 侧已算好 delta，显式传入 ARGV[2...]
-- 2) 兼容模式：不传 ARGV[2...]，脚本回退读取 incKey(HGETALL)
-- 这样可以兼容历史数据格式，同时保证新公式可控地由 Go 统一计算。
local items = nil
if ARGV ~= nil and #ARGV >= 3 and ((#ARGV - 1) % 2 == 0) then
    items = {}
    for i = 2, #ARGV do
        table.insert(items, ARGV[i])
    end
else
    items = redis.call('HGETALL', incKey)
end

local merged = 0
if items ~= nil and #items > 0 then
    for i = 1, #items, 2 do
        local member = items[i]
        local delta = tonumber(items[i + 1])
        if member ~= nil and member ~= '' and delta ~= nil and delta ~= 0 then
            -- 统一保留固定小数位，避免浮点噪声导致 zset 分值不可控。
            if precision > 0 then
                local factor = 10 ^ precision
                delta = math.floor(delta * factor + 0.5) / factor
            end
            redis.call('ZINCRBY', zsetKey, delta, member)
            merged = merged + 1
        end
    end
end

-- 合并后立刻删除增量桶，保证“消费一次且仅一次”。
redis.call('DEL', incKey)
return {merged}
