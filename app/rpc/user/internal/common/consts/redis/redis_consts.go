package redis

import "strconv"

const (
	// RedisUserSessionPrefix 用户登录态 token 前缀 user:session:{token}
	RedisUserSessionPrefix = "user:session"
	// RedisUserSessionUserPrefix 用户登录态 userId 前缀 user:session:user:{userId}
	RedisUserSessionUserPrefix = "user:session:user"
	// RedisUserSessionExpireSecondsDefault 用户登录态默认过期时间：7天
	RedisUserSessionExpireSecondsDefault = 7 * 24 * 60 * 60
)

func GetRedisPrefixKey(prefix string, id string) string {
	return prefix + ":" + id
}

func BuildUserSessionKey(token string) string {
	return GetRedisPrefixKey(RedisUserSessionPrefix, token)
}

func BuildUserSessionUserKey(userID int64) string {
	return GetRedisPrefixKey(RedisUserSessionUserPrefix, strconv.FormatInt(userID, 10))
}
