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
	renewRatio        = 1.0 / 3.0
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
	if strings.EqualFold(authorization, "Bearer") {
		return "", false
	}

	return authorization, true
}

func verifyAndRenewSession(ctx context.Context, rds *redis.Redis, token string, ttl time.Duration) (int64, error) {
	if token == "" {
		return 0, errorx.NewUnauthorized("用户未登录")
	}

	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds <= 0 {
		ttlSeconds = int(defaultSessionTTL.Seconds())
	}
	threshold := int(float64(ttlSeconds) * renewRatio)
	if threshold <= 0 {
		threshold = 1
	}

	tokenKey := redisTokenKeyPrefix + ":" + token
	resp, err := rds.EvalCtx(ctx, VerifyAndRenewSessionScript, []string{tokenKey}, token, redisUserKeyPrefix, ttlSeconds, threshold)
	if err != nil {
		return 0, err
	}

	var userIDStr string
	switch v := resp.(type) {
	case string:
		userIDStr = v
	case []byte:
		userIDStr = string(v)
	default:
		userIDStr = ""
	}
	userIDStr = strings.TrimSpace(userIDStr)
	if userIDStr == "" {
		return 0, errorx.NewUnauthorized("用户未登录")
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		return 0, errorx.NewUnauthorized("用户未登录")
	}

	return userID, nil
}
