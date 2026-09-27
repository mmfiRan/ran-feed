---@diagnostic disable: undefined-global
-- 热榜裁剪 + 快照重建脚本（原子执行）
-- 把主榜裁剪到候选池大小、构建对外快照、切 latest 指针合并为一个原子操作，
-- 避免裁剪与快照之间存在非原子窗口导致主榜与快照口径不一致。
-- 快更与冷更共用：两者重建主榜后都在此统一裁剪，保证主榜恒为候选池大小不变量。
--
-- 主榜（候选池）裁到 pruneN，远大于对外快照 snapN：
-- 多出的 pruneN-snapN 条是“候补席”，分数够但未进对外榜，
-- 用来扛删除/封禁挖空座位时的补位——下架删一个、候补自动顶一个，
-- 对外快照永远满 snapN，不会被删除侵蚀瘪掉。两者相等则无候补垫，删一条瘪一条。
--
-- KEYS[1] = 热榜 zset key（主榜 候选池）
-- KEYS[2] = 快照 zset key
-- KEYS[3] = 最新快照 id key
-- ARGV[1] = pruneN 主榜候选池大小 裁剪线
-- ARGV[2] = snapN  对外快照大小
-- ARGV[3] = snapshotId
-- ARGV[4] = ttlSeconds
-- 返回：{count} 快照实际写入条数

local zsetKey = KEYS[1]
local snapshotKey = KEYS[2]
local latestKey = KEYS[3]
local pruneN = tonumber(ARGV[1])
local snapN = tonumber(ARGV[2])
local snapshotId = ARGV[3]
local ttlSeconds = tonumber(ARGV[4])

if pruneN == nil or pruneN <= 0 then
    return {0}
end
-- snapN 缺省或越界时退化为 pruneN 保证快照不会比主榜还大
if snapN == nil or snapN <= 0 or snapN > pruneN then
    snapN = pruneN
end

-- 1 裁剪主榜到候选池大小 pruneN：ZREMRANGEBYRANK 按升序 rank 删除 [0, -(pruneN+1)]，
--   即移除分数最低的多余成员，仅保留分数最高的 pruneN 名。成员数 <= pruneN 时为空操作。
redis.call('ZREMRANGEBYRANK', zsetKey, 0, -(pruneN + 1))

-- 2 取主榜前 snapN 构建对外快照
local raw = redis.call('ZREVRANGE', zsetKey, 0, snapN - 1, 'WITHSCORES')
if raw == nil or #raw == 0 then
    return {0}
end

redis.call('DEL', snapshotKey)
local count = 0
for i = 1, #raw, 2 do
    local member = raw[i]
    local score = raw[i + 1]
    if member ~= nil and member ~= '' and score ~= nil then
        redis.call('ZADD', snapshotKey, score, member)
        count = count + 1
    end
end

if ttlSeconds ~= nil and ttlSeconds > 0 then
    redis.call('EXPIRE', snapshotKey, ttlSeconds)
end

if snapshotId ~= nil and snapshotId ~= '' then
    redis.call('SET', latestKey, snapshotId)
end

return {count}