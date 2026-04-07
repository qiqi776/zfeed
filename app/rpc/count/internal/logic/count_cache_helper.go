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

func buildUserProfileCountsCacheKey(userID int64) string {
	return redisconsts.BuildUserProfileCountsKey(userID)
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
