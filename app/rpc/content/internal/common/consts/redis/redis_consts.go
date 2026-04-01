package redis

import "fmt"

const (
	RedisUserPublishPrefix 		= "user:publish"
	RedisUserPublishKeepLatestN = 5000
)

func BuildUserPublishKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisUserPublishPrefix, userID)
}