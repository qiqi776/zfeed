// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"zfeed/app/front/internal/config"
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
		// Auth validation will be implemented later. Keep the skeleton dependency-aware
		// so middleware construction matches the real runtime wiring.
		_ = m.redis
		_ = m.config
		next(w, r)
	}
}
