package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/front/internal/config"
)

func TestExtractToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{
			name:      "missing header",
			header:    "",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "bearer token",
			header:    "Bearer abc",
			wantToken: "abc",
			wantOK:    true,
		},
		{
			name:      "raw token",
			header:    "abc",
			wantToken: "abc",
			wantOK:    true,
		},
		{
			name:      "blank bearer token",
			header:    "Bearer   ",
			wantToken: "",
			wantOK:    false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotToken, gotOK := extractToken(tc.header)
			if gotToken != tc.wantToken || gotOK != tc.wantOK {
				t.Fatalf("extractToken(%q) = (%q, %v), want (%q, %v)", tc.header, gotToken, gotOK, tc.wantToken, tc.wantOK)
			}
		})
	}
}

func TestVerifyAndRenewSession(t *testing.T) {
	t.Parallel()

	t.Run("renews ttl when below threshold", func(t *testing.T) {
		t.Parallel()

		store, client := newTestRedis(t)
		const (
			token  = "renew-token"
			userID = int64(1001)
		)
		seedSession(t, store, token, userID, 2*time.Second)

		gotUserID, err := verifyAndRenewSession(context.Background(), client, token, 9*time.Second)
		if err != nil {
			t.Fatalf("verifyAndRenewSession returned error: %v", err)
		}
		if gotUserID != userID {
			t.Fatalf("verifyAndRenewSession userID = %d, want %d", gotUserID, userID)
		}

		tokenTTL := store.TTL(buildTokenKey(token))
		userTTL := store.TTL(buildUserKey(userID))
		if tokenTTL != 9*time.Second {
			t.Fatalf("token ttl = %v, want %v", tokenTTL, 9*time.Second)
		}
		if userTTL != 9*time.Second {
			t.Fatalf("user ttl = %v, want %v", userTTL, 9*time.Second)
		}
	})

	t.Run("does not renew ttl when above threshold", func(t *testing.T) {
		t.Parallel()

		store, client := newTestRedis(t)
		const (
			token  = "keep-token"
			userID = int64(1002)
		)
		seedSession(t, store, token, userID, 8*time.Second)

		gotUserID, err := verifyAndRenewSession(context.Background(), client, token, 9*time.Second)
		if err != nil {
			t.Fatalf("verifyAndRenewSession returned error: %v", err)
		}
		if gotUserID != userID {
			t.Fatalf("verifyAndRenewSession userID = %d, want %d", gotUserID, userID)
		}

		tokenTTL := store.TTL(buildTokenKey(token))
		userTTL := store.TTL(buildUserKey(userID))
		if tokenTTL != 8*time.Second {
			t.Fatalf("token ttl = %v, want %v", tokenTTL, 8*time.Second)
		}
		if userTTL != 8*time.Second {
			t.Fatalf("user ttl = %v, want %v", userTTL, 8*time.Second)
		}
	})

	t.Run("fails when token mismatch", func(t *testing.T) {
		t.Parallel()

		store, client := newTestRedis(t)
		const (
			token       = "old-token"
			currentUser = int64(1003)
		)
		seedSession(t, store, token, currentUser, 5*time.Second)
		if err := store.Set(buildUserKey(currentUser), "new-token"); err != nil {
			t.Fatalf("override user key: %v", err)
		}
		store.SetTTL(buildUserKey(currentUser), 5*time.Second)

		_, err := verifyAndRenewSession(context.Background(), client, token, 9*time.Second)
		if err == nil {
			t.Fatal("verifyAndRenewSession error = nil, want not nil")
		}
	})
}

func TestUserLoginStatusAuthMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("injects context for valid session", func(t *testing.T) {
		t.Parallel()

		store, client := newTestRedis(t)
		const (
			token  = "auth-token"
			userID = int64(2001)
		)
		seedSession(t, store, token, userID, 5*time.Second)

		mw := NewUserLoginStatusAuthMiddleware(client, config.Config{SessionTTL: 30})
		called := false

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(headerAuthorization, "Bearer "+token)
		rec := httptest.NewRecorder()

		mw.Handle(func(w http.ResponseWriter, r *http.Request) {
			called = true

			gotUserID, ok := r.Context().Value(ctxKeyUserID).(int64)
			if !ok || gotUserID != userID {
				t.Fatalf("context user_id = %v, want %d", r.Context().Value(ctxKeyUserID), userID)
			}
			gotToken, ok := r.Context().Value(ctxKeyToken).(string)
			if !ok || gotToken != token {
				t.Fatalf("context token = %v, want %q", r.Context().Value(ctxKeyToken), token)
			}
		})(rec, req)

		if !called {
			t.Fatal("next handler was not called")
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		t.Parallel()

		_, client := newTestRedis(t)
		mw := NewUserLoginStatusAuthMiddleware(client, config.Config{SessionTTL: 30})
		called := false

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()

		mw.Handle(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})(rec, req)

		if called {
			t.Fatal("next handler was called for missing token")
		}
	})
}

func TestOptionalLoginMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("passes through without token", func(t *testing.T) {
		t.Parallel()

		_, client := newTestRedis(t)
		mw := NewOptionalLoginMiddleware(client, config.Config{SessionTTL: 30})
		called := false

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		rec := httptest.NewRecorder()

		mw.Handle(func(w http.ResponseWriter, r *http.Request) {
			called = true
			if r.Context().Value(ctxKeyUserID) != nil {
				t.Fatalf("unexpected user_id in context: %v", r.Context().Value(ctxKeyUserID))
			}
		})(rec, req)

		if !called {
			t.Fatal("next handler was not called")
		}
	})

	t.Run("injects context for valid token", func(t *testing.T) {
		t.Parallel()

		store, client := newTestRedis(t)
		const (
			token  = "optional-token"
			userID = int64(2002)
		)
		seedSession(t, store, token, userID, 5*time.Second)

		mw := NewOptionalLoginMiddleware(client, config.Config{SessionTTL: 30})
		called := false

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set(headerAuthorization, token)
		rec := httptest.NewRecorder()

		mw.Handle(func(w http.ResponseWriter, r *http.Request) {
			called = true
			gotUserID, ok := r.Context().Value(ctxKeyUserID).(int64)
			if !ok || gotUserID != userID {
				t.Fatalf("context user_id = %v, want %d", r.Context().Value(ctxKeyUserID), userID)
			}
		})(rec, req)

		if !called {
			t.Fatal("next handler was not called")
		}
	})

	t.Run("passes through invalid token without context", func(t *testing.T) {
		t.Parallel()

		_, client := newTestRedis(t)
		mw := NewOptionalLoginMiddleware(client, config.Config{SessionTTL: 30})
		called := false

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set(headerAuthorization, "invalid-token")
		rec := httptest.NewRecorder()

		mw.Handle(func(w http.ResponseWriter, r *http.Request) {
			called = true
			if r.Context().Value(ctxKeyUserID) != nil {
				t.Fatalf("unexpected user_id in context: %v", r.Context().Value(ctxKeyUserID))
			}
		})(rec, req)

		if !called {
			t.Fatal("next handler was not called")
		}
	})
}

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	return store, client
}

func seedSession(t *testing.T, store *miniredis.Miniredis, token string, userID int64, ttl time.Duration) {
	t.Helper()

	if err := store.Set(buildTokenKey(token), int64ToString(userID)); err != nil {
		t.Fatalf("seed token key value: %v", err)
	}
	if err := store.Set(buildUserKey(userID), token); err != nil {
		t.Fatalf("seed user key: %v", err)
	}
	store.SetTTL(buildTokenKey(token), ttl)
	store.SetTTL(buildUserKey(userID), ttl)
}

func buildTokenKey(token string) string {
	return redisTokenKeyPrefix + ":" + token
}

func buildUserKey(userID int64) string {
	return redisUserKeyPrefix + ":" + int64ToString(userID)
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
