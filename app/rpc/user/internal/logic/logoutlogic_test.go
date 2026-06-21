package logic

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"zfeed/app/rpc/user/internal/common/consts/redis"
	"zfeed/app/rpc/user/internal/common/utils/session"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
)

func TestLogoutRejectsInvalidParams(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		token  string
	}{
		{name: "zero user", userID: 0, token: "token"},
		{name: "negative user", userID: -1, token: "token"},
		{name: "blank token", userID: 1001, token: " \t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logic := NewLogoutLogic(context.Background(), &svc.ServiceContext{})

			_, err := logic.Logout(&user.LogoutReq{
				UserId: tt.userID,
				Token:  tt.token,
			})
			if err == nil {
				t.Fatal("expected invalid logout params to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
		})
	}
}

func TestLogoutKeepsOtherSession(t *testing.T) {
	tests := []struct {
		name                  string
		requestUserID         int64
		requestUserHasSession bool
	}{
		{
			name:          "request user without session",
			requestUserID: 3102,
		},
		{
			name:                  "request user with session",
			requestUserID:         3103,
			requestUserHasSession: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const (
				ownerUserID  = 3101
				ownerToken   = "owner-token"
				requestToken = "request-token"
			)
			store, redisClient := newUserLogicTestRedisStore(t)
			if err := session.SaveSession(context.Background(), redisClient, ownerUserID, ownerToken, time.Minute); err != nil {
				t.Fatalf("seed owner session: %v", err)
			}
			if tt.requestUserHasSession {
				if err := session.SaveSession(context.Background(), redisClient, tt.requestUserID, requestToken, time.Minute); err != nil {
					t.Fatalf("seed request user session: %v", err)
				}
			}

			logic := NewLogoutLogic(context.Background(), &svc.ServiceContext{
				Redis: redisClient,
			})
			if _, err := logic.Logout(&user.LogoutReq{
				UserId: tt.requestUserID,
				Token:  ownerToken,
			}); err != nil {
				t.Fatalf("logout with mismatched token: %v", err)
			}

			ownerTokenKey := redis.BuildUserSessionKey(ownerToken)
			if !store.Exists(ownerTokenKey) {
				t.Fatalf("owner token key %q was deleted", ownerTokenKey)
			}
			ownerUserKey := redis.BuildUserSessionUserKey(ownerUserID)
			if got, err := store.Get(ownerUserKey); err != nil || got != ownerToken {
				t.Fatalf("owner user key = %q, %v; want %q", got, err, ownerToken)
			}
			if tt.requestUserHasSession {
				requestTokenKey := redis.BuildUserSessionKey(requestToken)
				if !store.Exists(requestTokenKey) {
					t.Fatalf("request user token key %q was deleted", requestTokenKey)
				}
				requestUserKey := redis.BuildUserSessionUserKey(tt.requestUserID)
				if got, err := store.Get(requestUserKey); err != nil || got != requestToken {
					t.Fatalf("request user key = %q, %v; want %q", got, err, requestToken)
				}
			}
		})
	}
}
