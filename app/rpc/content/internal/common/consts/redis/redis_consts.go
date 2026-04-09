package redis

import "fmt"

const (
	RedisUserPublishPrefix      = "feed:user:publish"
	RedisFollowInboxPrefix      = "feed:follow:inbox"
	RedisFollowInboxLockPrefix  = "feed:follow:inbox:lock"
	RedisUserPublishKeepLatestN = 5000
	RedisFollowInboxKeepLatestN = 5000
)

func BuildUserPublishKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisUserPublishPrefix, userID)
}

func BuildUserPublishFeedKey(userID int64) string {
	return BuildUserPublishKey(userID)
}

func BuildFollowInboxKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisFollowInboxPrefix, userID)
}

func BuildFollowInboxRebuildLockKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisFollowInboxLockPrefix, userID)
}

func BuildFollowInboxLockKey(userID int64) string {
	return BuildFollowInboxRebuildLockKey(userID)
}
