// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/front/internal/config"
)

type OptionalLoginMiddleware struct {
	redis  *redis.Redis
	config config.Config
}

func NewOptionalLoginMiddleware(rds *redis.Redis, cfg config.Config) *OptionalLoginMiddleware {
	return &OptionalLoginMiddleware{
		redis:  rds,
		config: cfg,
	}
}

func (m *OptionalLoginMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r.Header.Get(headerAuthorization))
		if !ok {
			next(w, r)
			return
		}

		userID, err := verifyAndRenewSession(r.Context(), m.redis, token, parseSessionTTL(m.config))
		if err != nil {
			next(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		ctx = context.WithValue(ctx, ctxKeyToken, token)
		next(w, r.WithContext(ctx))
	}
}
