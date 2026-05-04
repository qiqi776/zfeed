package redis

import "strconv"

const (
	CountValuePrefix               = "count:value"
	CountValueExpireSeconds        = 24 * 60 * 60
	UserProfileCountsPrefix        = "count:user:profile"
	UserProfileCountsExpireSeconds = 24 * 60 * 60
	CountRebuildLockPrefix         = "lock:rebuild:count"
	ProfileRebuildLockPrefix       = "lock:rebuild:count:user:profile"
	HotFeedIncPrefix               = "feed:hot:global:inc"
	HotFeedIncShards               = 64
)

func GetRedisPrefixKey(prefix, id string) string {
	return prefix + ":" + id
}

func BuildCountValueKey(bizType string, targetType string, targetID string) string {
	return GetRedisPrefixKey(GetRedisPrefixKey(GetRedisPrefixKey(CountValuePrefix, bizType), targetType), targetID)
}

func BuildUserProfileCountsKey(userID int64) string {
	return GetRedisPrefixKey(UserProfileCountsPrefix, strconv.FormatInt(userID, 10))
}

func BuildHotFeedIncKey(shard int) string {
	return GetRedisPrefixKey(HotFeedIncPrefix, strconv.Itoa(shard))
}
