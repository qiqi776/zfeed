// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
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
		// Login probing will be implemented later. Keep the skeleton dependency-aware
		// so ServiceContext already wires Redis and config into the middleware.
		_ = m.redis
		_ = m.config
		next(w, r)
	}
}
