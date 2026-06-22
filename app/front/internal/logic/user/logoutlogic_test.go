package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"
)

func TestLogoutForwards(t *testing.T) {
	stub := &stubUserService{logout: &userservice.LogoutRes{}}
	ctx := context.WithValue(context.Background(), "user_id", int64(3001))
	ctx = context.WithValue(ctx, "token", "session-token")
	logic := NewLogoutLogic(ctx, &svc.ServiceContext{UserRpc: stub})

	if _, err := logic.Logout(); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if stub.lastLogoutReq == nil {
		t.Fatal("expected logout request to be forwarded")
	}
	if stub.lastLogoutReq.GetUserId() != 3001 || stub.lastLogoutReq.GetToken() != "session-token" {
		t.Fatalf("unexpected logout request: %+v", stub.lastLogoutReq)
	}
}

func TestLogoutRejectsBlankToken(t *testing.T) {
	stub := &stubUserService{logout: &userservice.LogoutRes{}}
	ctx := context.WithValue(context.Background(), "user_id", int64(3001))
	ctx = context.WithValue(ctx, "token", " \t\n ")
	logic := NewLogoutLogic(ctx, &svc.ServiceContext{UserRpc: stub})

	_, err := logic.Logout()
	if err == nil {
		t.Fatal("expected blank token to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusUnauthorized {
		t.Fatalf("error = %v, want unauthorized", err)
	}
	if stub.lastLogoutReq != nil {
		t.Fatalf("blank token should not call UserRpc, got %+v", stub.lastLogoutReq)
	}
}

func TestLogoutRPCError(t *testing.T) {
	wantErr := errors.New("logout rpc failed")
	ctx := context.WithValue(context.Background(), "user_id", int64(3001))
	ctx = context.WithValue(ctx, "token", "session-token")
	logic := NewLogoutLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{err: wantErr},
	})

	_, err := logic.Logout()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestLogoutNilRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(3001))
	ctx = context.WithValue(ctx, "token", "session-token")
	logic := NewLogoutLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{},
	})

	_, err := logic.Logout()
	if err == nil {
		t.Fatal("expected nil logout rpc response to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusInternalServerError {
		t.Fatalf("error = %v, want internal server error", err)
	}
}
