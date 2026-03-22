---@diagnostic disable: undefined-global
-- Redis登录态删除Lua脚本
-- KEYS[1]=tokenKey
-- KEYS[2]=userKey
-- ARGV[1]=token
-- 返回: 1

local curToken = redis.call("GET", KEYS[2])
if curToken == ARGV[1] then
  redis.call("DEL", KEYS[2])
end
redis.call("DEL", KEYS[1])
return 1
