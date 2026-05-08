package likelogic

import (
	"context"
	"strconv"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
)

const (
	likeCacheValueLiked   = "1"
	likeCacheValueUnliked = "0"
)

func likeCacheKey(userID int64) string {
	return rediskey.BuildLikeUserKey(strconv.FormatInt(userID, 10))
}

// likeTargetKey 生成点赞哈希表中的 field
// 格式: "scene:content_id"
func likeTargetKey(scene interaction.Scene, contentID int64) string {
	return strconv.FormatInt(int64(scene), 10) + ":" + strconv.FormatInt(contentID, 10)
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

func cacheLikeState(ctx context.Context, rds *gzredis.Redis, userLikeKey, field string, isLiked bool) error {
	cacheValue := likeCacheValueUnliked
	if isLiked {
		cacheValue = likeCacheValueLiked
	}
	if err := rds.HsetCtx(ctx, userLikeKey, field, cacheValue); err != nil {
		return err
	}
	return rds.ExpireCtx(ctx, userLikeKey, rediskey.LikeExpireSeconds)
}
