package redis

const (
	RedisLikeUserPrefix    = "like:user"
	RedisLikeExpireSeconds = 5 * 24 * 60 * 60
)

func BuildLikeUserKey(userID string) string {
	return RedisLikeUserPrefix + ":" + userID
}
