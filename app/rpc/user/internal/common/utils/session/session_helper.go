package session

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "zfeed/app/rpc/user/internal/common/consts/redis"
	luautils "zfeed/app/rpc/user/internal/common/utils/lua"
	"zfeed/app/rpc/user/internal/config"
)

func GetSessionTTL(cfg config.Config) time.Duration {
	sessionTTL := time.Duration(cfg.SessionTTL) * time.Second
	if sessionTTL <= 0 {
		return time.Duration(rediskey.RedisUserSessionExpireSecondsDefault) * time.Second
	}
	return sessionTTL
}

func NewSessionToken() string {
	return uuid.NewString()
}

func SaveSession(ctx context.Context, r *redis.Redis, userID int64, token string, ttl time.Duration) error {
	tokenKey := rediskey.BuildUserSessionKey(token)
	userKey := rediskey.BuildUserSessionUserKey(userID)

	_, err := r.EvalCtx(
		ctx,
		luautils.SaveSessionScript,
		[]string{tokenKey, userKey},
		strconv.FormatInt(userID, 10),
		token,
		int(ttl.Seconds()),
		rediskey.RedisUserSessionPrefix,
	)
	return err
}

func RemoveSession(ctx context.Context, r *redis.Redis, userID int64, token string) error {
	tokenKey := rediskey.BuildUserSessionKey(token)
	userKey := rediskey.BuildUserSessionUserKey(userID)

	_, err := r.EvalCtx(
		ctx,
		luautils.RemoveSessionScript,
		[]string{tokenKey, userKey},
		token,
	)
	return err
}
