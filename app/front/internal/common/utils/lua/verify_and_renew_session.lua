---@diagnostic disable: undefined-global
-- 登录态校验 + 续期 Lua 脚本（tokenKey + userKey，原子性）
-- KEYS[1]=tokenKey (user:session:{token})
-- ARGV[1]=token
-- ARGV[2]=userKey前缀 (user:session:user)
-- ARGV[3]=ttl(秒)
-- ARGV[4]=续期阈值(秒)，当 tokenKey 的剩余 TTL 小于该值才续期
-- 返回: userId 字符串；失败返回空字符串

local userId = redis.call("GET", KEYS[1])
if not userId or userId == "" then
  return ""
end

local userKey = ARGV[2] .. ":" .. userId
local token = redis.call("GET", userKey)
if not token or token == "" or token ~= ARGV[1] then
  return ""
end

local ttl = redis.call("TTL", KEYS[1])
if ttl and ttl >= 0 and ttl < tonumber(ARGV[4]) then
  redis.call("EXPIRE", KEYS[1], tonumber(ARGV[3]))
  redis.call("EXPIRE", userKey, tonumber(ARGV[3]))
end

return userId
