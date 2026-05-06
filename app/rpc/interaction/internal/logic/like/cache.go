package likelogic

import (
	"context"
	"strconv"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
)

const (
	likeCacheValueLiked   = "1"
	likeCacheValueUnliked = "0"
)

func likeCacheKey(userID int64) string {
	return rediskey.BuildLikeUserKey(strconv.FormatInt(userID, 10))
}

func parseLikeCacheValue(value string) (isLiked bool, ok bool) {
	switch value {
	case likeCacheValueLiked:
		return true, true
	case likeCacheValueUnliked:
		return false, true
	default:
		return false, false
	}
}

func cacheLikeState(ctx context.Context, rds *gzredis.Redis, userLikeKey, contentID string, isLiked bool) error {
	cacheValue := likeCacheValueUnliked
	if isLiked {
		cacheValue = likeCacheValueLiked
	}

	if err := rds.HsetCtx(ctx, userLikeKey, contentID, cacheValue); err != nil {
		return err
	}

	return rds.ExpireCtx(ctx, userLikeKey, rediskey.LikeExpireSeconds)
}
