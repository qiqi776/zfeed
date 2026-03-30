package redis

import "strconv"

const (
	RedisUserSessionPrefix               = "user:session"
	RedisUserSessionUserPrefix           = "user:session:user"
	RedisUserSessionExpireSecondsDefault = 7 * 24 * 60 * 60
)

func getRedisPrefixKey(prefix, id string) string {
	return prefix + ":" + id
}

func BuildUserSessionKey(token string) string {
	return getRedisPrefixKey(RedisUserSessionPrefix, token)
}

func BuildUserSessionUserKey(userID int64) string {
	return getRedisPrefixKey(RedisUserSessionUserPrefix, strconv.FormatInt(userID, 10))
}
