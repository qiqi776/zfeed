// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"

	"zfeed/app/front/internal/config"
	"zfeed/pkg/errorx"
)

type UserLoginStatusAuthMiddleware struct {
	redis  *redis.Redis
	config config.Config
}

func NewUserLoginStatusAuthMiddleware(rds *redis.Redis, cfg config.Config) *UserLoginStatusAuthMiddleware {
	return &UserLoginStatusAuthMiddleware{
		redis:  rds,
		config: cfg,
	}
}

func (m *UserLoginStatusAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r.Header.Get(headerAuthorization))
		if !ok {
			httpx.ErrorCtx(r.Context(), w, errorx.NewUnauthorized("用户未登录"))
			return
		}

		userID, err := verifyAndRenewSession(r.Context(), m.redis, token, parseSessionTTL(m.config))
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewUnauthorized("用户未登录"))
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		ctx = context.WithValue(ctx, ctxKeyToken, token)
		next(w, r.WithContext(ctx))
	}
}
