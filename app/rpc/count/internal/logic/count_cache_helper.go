package logic

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"zfeed/app/rpc/count/count"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
)

const cacheExpireJitterMaxSeconds = 600

const (
	rebuildLockExpireSeconds = 30
)

type cacheQueryResult int

const (
	cacheHit cacheQueryResult = iota
	cacheMiss
	cacheError
)

type userProfileCountsCache struct {
	FollowingCount int64 `json:"following_count"`
	FollowedCount  int64 `json:"followed_count"`
	LikeCount      int64 `json:"like_count"`
	FavoriteCount  int64 `json:"favorite_count"`
}

func buildCountValueCacheKey(bizType count.BizType, targetType count.TargetType, targetID int64) string {
	return redisconsts.BuildCountValueKey(
		strconv.FormatInt(int64(bizType), 10),
		strconv.FormatInt(int64(targetType), 10),
		strconv.FormatInt(targetID, 10),
	)
}

func buildCountValueMapKey(bizType count.BizType, targetType count.TargetType, targetID int64) string {
	return fmt.Sprintf("%d:%d:%d", bizType, targetType, targetID)
}

func buildCountValueRebuildLockKey(bizType count.BizType, targetType count.TargetType, targetID int64) string {
	return redisconsts.GetRedisPrefixKey(
		redisconsts.RedisCountRebuildLockPrefix,
		fmt.Sprintf("%d:%d:%d", bizType, targetType, targetID),
	)
}

func buildUserProfileCountsCacheKey(userID int64) string {
	return redisconsts.BuildUserProfileCountsKey(userID)
}

func buildUserProfileCountsRebuildLockKey(userID int64) string {
	return redisconsts.GetRedisPrefixKey(redisconsts.RedisUserProfileCountsRebuildLockPref, strconv.FormatInt(userID, 10))
}

func countCacheExpireSecondsWithJitter(base int) int {
	return base + rand.Intn(cacheExpireJitterMaxSeconds+1)
}

func marshalUserProfileCounts(value *count.GetUserProfileCountsRes) (string, error) {
	payload, err := json.Marshal(userProfileCountsCache{
		FollowingCount: value.GetFollowingCount(),
		FollowedCount:  value.GetFollowedCount(),
		LikeCount:      value.GetLikeCount(),
		FavoriteCount:  value.GetFavoriteCount(),
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func unmarshalUserProfileCounts(payload string) (*count.GetUserProfileCountsRes, error) {
	var cached userProfileCountsCache
	if err := json.Unmarshal([]byte(payload), &cached); err != nil {
		return nil, err
	}
	return &count.GetUserProfileCountsRes{
		FollowingCount: cached.FollowingCount,
		FollowedCount:  cached.FollowedCount,
		LikeCount:      cached.LikeCount,
		FavoriteCount:  cached.FavoriteCount,
	}, nil
}
