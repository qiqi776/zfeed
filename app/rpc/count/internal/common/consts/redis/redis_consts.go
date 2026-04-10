package redis

import "strconv"

const (
	RedisCountValuePrefix                 = "count:value"
	RedisCountValueExpireSeconds          = 24 * 60 * 60
	RedisUserProfileCountsPrefix          = "count:user:profile"
	RedisUserProfileCountsExpireSeconds   = 24 * 60 * 60
	RedisCountRebuildLockPrefix           = "lock:rebuild:count"
	RedisUserProfileCountsRebuildLockPref = "lock:rebuild:count:user:profile"
	RedisFeedHotGlobalIncPrefix           = "feed:hot:global:inc"
	RedisFeedHotIncDefaultShards          = 64
)

func GetRedisPrefixKey(prefix, id string) string {
	return prefix + ":" + id
}

func BuildCountValueKey(bizType string, targetType string, targetID string) string {
	return GetRedisPrefixKey(GetRedisPrefixKey(GetRedisPrefixKey(RedisCountValuePrefix, bizType), targetType), targetID)
}

func BuildUserProfileCountsKey(userID int64) string {
	return GetRedisPrefixKey(RedisUserProfileCountsPrefix, strconv.FormatInt(userID, 10))
}

func BuildHotFeedIncKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotGlobalIncPrefix, strconv.Itoa(shard))
}
