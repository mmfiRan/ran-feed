---@diagnostic disable: undefined-global
-- Redis登录态写入Lua脚本
-- KEYS[1]=tokenKey
-- KEYS[2]=userKey
-- ARGV[1]=userId
-- ARGV[2]=token
-- ARGV[3]=ttl(秒)
-- ARGV[4]=tokenKey前缀
-- 返回: 1

local oldToken = redis.call("GET", KEYS[2])
if oldToken and oldToken ~= "" then
  redis.call("DEL", ARGV[4] .. ":" .. oldToken)
end
redis.call("SETEX", KEYS[1], ARGV[3], ARGV[1])
redis.call("SETEX", KEYS[2], ARGV[3], ARGV[2])
return 1
