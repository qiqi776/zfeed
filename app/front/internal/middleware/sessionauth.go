package middleware

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/front/internal/config"
	"zfeed/pkg/errorx"
)

const (
	ctxKeyUserID        = "user_id"
	ctxKeyToken         = "token"
	headerAuthorization = "Authorization"

	redisTokenKeyPrefix = "user:session"
	redisUserKeyPrefix  = "user:session:user"

	defaultSessionTTL = 7 * 24 * time.Hour
)

func parseSessionTTL(cfg config.Config) time.Duration {
	if cfg.SessionTTL <= 0 {
		return defaultSessionTTL
	}
	return time.Duration(cfg.SessionTTL) * time.Second
}

func extractToken(authHeader string) (string, bool) {
	authorization := strings.TrimSpace(authHeader)
	if authorization == "" {
		return "", false
	}

	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		token := strings.TrimSpace(parts[1])
		return token, token != ""
	}

	return authorization, true
}

func verifyAndRenewSession(ctx context.Context, rds *redis.Redis, token string, ttl time.Duration) (int64, error) {
	if token == "" {
		return 0, errorx.NewMsg("用户未登录")
	}

	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds <= 0 {
		ttlSeconds = int(defaultSessionTTL.Seconds())
	}

	tokenKey := redisTokenKeyPrefix + ":" + token
	userIDStr, err := rds.GetCtx(ctx, tokenKey)
	if err != nil || strings.TrimSpace(userIDStr) == "" {
		return 0, errorx.NewMsg("用户未登录")
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		return 0, errorx.NewMsg("用户未登录")
	}

	userKey := redisUserKeyPrefix + ":" + strconv.FormatInt(userID, 10)
	savedToken, err := rds.GetCtx(ctx, userKey)
	if err != nil || savedToken != token {
		return 0, errorx.NewMsg("用户未登录")
	}

	if err = rds.ExpireCtx(ctx, tokenKey, ttlSeconds); err != nil {
		return 0, err
	}
	if err = rds.ExpireCtx(ctx, userKey, ttlSeconds); err != nil {
		return 0, err
	}

	return userID, nil
}
