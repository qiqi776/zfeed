package redis

import "fmt"

const (
	RedisFeedHotGlobalKey            = "feed:hot:global"
	RedisFeedHotGlobalLatestKey      = "feed:hot:global:latest"
	RedisFeedHotGlobalSnapshotPrefix = "feed:hot:global:snap"
	RedisFeedHotGlobalIncPrefix      = "feed:hot:global:inc"
	RedisFeedHotIncDefaultShards     = 64
	RedisFeedHotFastLockPrefix       = "feed:hot:global:lock:fast"
	RedisFeedHotColdLockPrefix       = "feed:hot:global:lock:cold"
	RedisFeedHotBucketCleanupPrefix  = "feed:hot:global:lock:cleanup"

	RedisUserPublishPrefix       = "feed:user:publish"
	RedisUserPublishLockPrefix   = "feed:user:publish:lock"
	RedisUserFavoritePrefix      = "feed:user:favorite"
	RedisUserFavoriteLockPrefix  = "feed:user:favorite:lock"
	RedisFollowInboxPrefix       = "feed:follow:inbox"
	RedisFollowInboxLockPrefix   = "feed:follow:inbox:lock"
	RedisUserPublishKeepLatestN  = 5000
	RedisUserFavoriteKeepLatestN = 5000
	RedisFollowInboxKeepLatestN  = 5000
)

func BuildUserPublishKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisUserPublishPrefix, userID)
}

func BuildHotFeedSnapshotKey(snapshotID string) string {
	return fmt.Sprintf("%s:%s", RedisFeedHotGlobalSnapshotPrefix, snapshotID)
}

func BuildHotFeedIncKey(shard int) string {
	return fmt.Sprintf("%s:%d", RedisFeedHotGlobalIncPrefix, shard)
}

func BuildHotFeedFastLockKey(bucket string) string {
	return fmt.Sprintf("%s:%s", RedisFeedHotFastLockPrefix, bucket)
}

func BuildHotFeedColdLockKey(date string) string {
	return fmt.Sprintf("%s:%s", RedisFeedHotColdLockPrefix, date)
}

func BuildHotFeedBucketCleanupLockKey(date string) string {
	return fmt.Sprintf("%s:%s", RedisFeedHotBucketCleanupPrefix, date)
}

func BuildUserPublishFeedKey(userID int64) string {
	return BuildUserPublishKey(userID)
}

func BuildUserPublishRebuildLockKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisUserPublishLockPrefix, userID)
}

func BuildUserFavoriteFeedKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisUserFavoritePrefix, userID)
}

func BuildUserFavoriteRebuildLockKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RedisUserFavoriteLockPrefix, userID)
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
