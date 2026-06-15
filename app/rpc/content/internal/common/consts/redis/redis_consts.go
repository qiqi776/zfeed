package redis

import "fmt"

const (
	HotFeedKey               = "feed:hot:global"
	HotFeedLatestKey         = "feed:hot:global:latest"
	HotFeedSnapshotPrefix    = "feed:hot:global:snap"
	HotFeedIncPrefix         = "feed:hot:global:inc"
	HotFeedIncShards         = 64
	HotFeedFastLockPrefix    = "feed:hot:global:lock:fast"
	HotFeedColdLockPrefix    = "feed:hot:global:lock:cold"
	HotFeedCleanupLockPrefix = "feed:hot:global:lock:cleanup"

	RecommendNewContentKey        = "feed:rec:new:global"
	RecommendNewContentMetaPrefix = "feed:rec:new:meta"
	RecommendUserProfilePrefix    = "rec:user:profile"
	RecommendContentTagsPrefix    = "rec:content:tags"
	RecommendTagIndexPrefix       = "rec:tag:index"
	RecommendSeenPrefix           = "rec:seen"
	RecommendSeenCountPrefix      = "rec:seen:count"
	RecommendCandidatePrefix      = "feed:rec:candidate"
	RecommendUserSnapshotPrefix   = "feed:rec:user:snap"
	RecommendUserSnapshotMeta     = "feed:rec:user:snapmeta"
	RecommendUserSnapshotSource   = "feed:rec:user:snapsource"
	RecommendNewCleanupLockPrefix = "feed:rec:new:lock:cleanup"
	RecommendTagRefreshLockPrefix = "rec:tag:index:lock:refresh"

	UserPublishPrefix      = "feed:user:publish"
	UserPublishLockPrefix  = "feed:user:publish:lock"
	UserFavoritePrefix     = "feed:user:favorite"
	UserFavoriteLockPrefix = "feed:user:favorite:lock"
	FollowInboxPrefix      = "feed:follow:inbox"
	FollowInboxLockPrefix  = "feed:follow:inbox:lock"
	FeedKeepLatestN        = 5000
)

func BuildUserPublishFeedKey(userID int64) string {
	return fmt.Sprintf("%s:%d", UserPublishPrefix, userID)
}

func BuildHotFeedSnapshotKey(snapshotID string) string {
	return fmt.Sprintf("%s:%s", HotFeedSnapshotPrefix, snapshotID)
}

func BuildHotFeedIncKey(shard int) string {
	return fmt.Sprintf("%s:%d", HotFeedIncPrefix, shard)
}

func BuildHotFeedFastLockKey(bucket string) string {
	return fmt.Sprintf("%s:%s", HotFeedFastLockPrefix, bucket)
}

func BuildHotFeedColdLockKey(date string) string {
	return fmt.Sprintf("%s:%s", HotFeedColdLockPrefix, date)
}

func BuildHotFeedBucketCleanupLockKey(date string) string {
	return fmt.Sprintf("%s:%s", HotFeedCleanupLockPrefix, date)
}

func BuildRecommendNewContentMetaKey(contentID int64) string {
	return fmt.Sprintf("%s:%d", RecommendNewContentMetaPrefix, contentID)
}

func BuildRecommendUserProfileKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RecommendUserProfilePrefix, userID)
}

func BuildRecommendContentTagsKey(contentID int64) string {
	return fmt.Sprintf("%s:%d", RecommendContentTagsPrefix, contentID)
}

func BuildRecommendTagIndexKey(tag string) string {
	return fmt.Sprintf("%s:%s", RecommendTagIndexPrefix, tag)
}

func BuildRecommendSeenKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RecommendSeenPrefix, userID)
}

func BuildRecommendSeenCountKey(userID int64) string {
	return fmt.Sprintf("%s:%d", RecommendSeenCountPrefix, userID)
}

func BuildRecommendUserSnapshotKey(snapshotID string) string {
	return fmt.Sprintf("%s:%s", RecommendUserSnapshotPrefix, snapshotID)
}

func BuildRecommendUserSnapshotMetaKey(snapshotID string) string {
	return fmt.Sprintf("%s:%s", RecommendUserSnapshotMeta, snapshotID)
}

func BuildRecommendUserSnapshotSourceKey(snapshotID string) string {
	return fmt.Sprintf("%s:%s", RecommendUserSnapshotSource, snapshotID)
}

func BuildRecommendNewCleanupLockKey(bucket string) string {
	return fmt.Sprintf("%s:%s", RecommendNewCleanupLockPrefix, bucket)
}

func BuildRecommendTagRefreshLockKey(bucket string) string {
	return fmt.Sprintf("%s:%s", RecommendTagRefreshLockPrefix, bucket)
}

func BuildUserPublishRebuildLockKey(userID int64) string {
	return fmt.Sprintf("%s:%d", UserPublishLockPrefix, userID)
}

func BuildUserFavoriteFeedKey(userID int64) string {
	return fmt.Sprintf("%s:%d", UserFavoritePrefix, userID)
}

func BuildUserFavoriteRebuildLockKey(userID int64) string {
	return fmt.Sprintf("%s:%d", UserFavoriteLockPrefix, userID)
}

func BuildFollowInboxKey(userID int64) string {
	return fmt.Sprintf("%s:%d", FollowInboxPrefix, userID)
}

func BuildFollowInboxLockKey(userID int64) string {
	return fmt.Sprintf("%s:%d", FollowInboxLockPrefix, userID)
}
