package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"

	"zfeed/app/admin-api/internal/config"
	"zfeed/pkg/errorx"
)

const (
	ctxKeyAdminUserID = "admin_user_id"
	ctxKeyAdminRole   = "admin_role"

	minAdminRole = 1 // moderator and above
)

// AdminAuthMiddleware validates the user session AND checks admin role.
type AdminAuthMiddleware struct {
	rds        *redis.Redis
	session    *UserLoginStatusAuthMiddleware
	sessionTTL time.Duration
}

func NewAdminAuthMiddleware(rds *redis.Redis, cfg config.Config) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{
		rds:        rds,
		session:    NewUserLoginStatusAuthMiddleware(rds, cfg),
		sessionTTL: parseSessionTTL(cfg),
	}
}

// Handle is the middleware entry point.
func (m *AdminAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r.Header.Get(headerAuthorization))
		if !ok || token == "" {
			httpx.ErrorCtx(r.Context(), w, errorx.NewUnauthorized("管理员未登录"))
			return
		}

		userID, err := verifyAndRenewSession(r.Context(), m.rds, token, m.sessionTTL)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewUnauthorized("管理员未登录"))
			return
		}

		role, err := m.fetchRole(r.Context(), userID)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("admin role fetch failed: user=%d err=%v", userID, err)
			httpx.ErrorCtx(r.Context(), w, errorx.NewUnauthorized("登录状态验证失败，请重新登录"))
			return
		}
		if role < minAdminRole {
			logx.WithContext(r.Context()).Infof("admin access denied: user=%d role=%d", userID, role)
			httpx.ErrorCtx(r.Context(), w, errorx.NewForbidden("无管理员权限"))
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyAdminUserID, userID)
		ctx = context.WithValue(ctx, ctxKeyAdminRole, role)

		next(w, r.WithContext(ctx))
	}
}

func (m *AdminAuthMiddleware) fetchRole(ctx context.Context, userID int64) (int32, error) {
	key := "user:role:" + strconv.FormatInt(userID, 10)
	val, err := m.rds.GetCtx(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("redis get role key %s: %w", key, err)
	}
	if val == "" {
		return 0, fmt.Errorf("admin role not found in cache for user %d", userID)
	}
	role, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse role value %q: %w", val, err)
	}
	return int32(role), nil
}

// GetAdminUserID extracts the admin user ID from context.
func GetAdminUserID(ctx context.Context) (int64, error) {
	uid, ok := ctx.Value(ctxKeyAdminUserID).(int64)
	if !ok || uid <= 0 {
		return 0, errors.New("admin user id not found in context")
	}
	return uid, nil
}

// GetAdminRole extracts the admin role from context.
func GetAdminRole(ctx context.Context) (int32, error) {
	role, ok := ctx.Value(ctxKeyAdminRole).(int32)
	if !ok {
		return 0, errors.New("admin role not found in context")
	}
	return role, nil
}
