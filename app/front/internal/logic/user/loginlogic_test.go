package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"
)

func TestLoginNormalizesMobileBeforeRPC(t *testing.T) {
	stub := &stubUserService{
		login: &userservice.LoginRes{
			UserId:    3002,
			Token:     "token",
			ExpiredAt: 456,
			Nickname:  "alice",
			Avatar:    "https://example.com/a.png",
		},
	}

	logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
		UserRpc: stub,
	})

	mobile := "+8613800000000"
	password := "123456"
	resp, err := logic.Login(&types.LoginReq{
		Mobile:   &mobile,
		Password: &password,
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if resp.UserId != 3002 || resp.Token != "token" || resp.ExpiredAt != 456 {
		t.Fatalf("unexpected login response: %+v", resp)
	}
	if stub.lastLoginReq == nil {
		t.Fatal("expected login request to be forwarded")
	}
	if stub.lastLoginReq.GetMobile() != "13800000000" || stub.lastLoginReq.GetPassword() != password {
		t.Fatalf("unexpected forwarded login req: %+v", stub.lastLoginReq)
	}
}

func TestLoginNilRPC(t *testing.T) {
	logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
		UserRpc: &stubUserService{},
	})

	mobile := "+8613800000000"
	password := "123456"
	_, err := logic.Login(&types.LoginReq{
		Mobile:   &mobile,
		Password: &password,
	})
	if err == nil {
		t.Fatal("expected nil login rpc response to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusInternalServerError {
		t.Fatalf("error = %v, want internal server error", err)
	}
}

func TestLoginRejectsBlankPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "empty", password: ""},
		{name: "spaces", password: "   "},
		{name: "tabs and newlines", password: "\t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubUserService{
				login: &userservice.LoginRes{UserId: 3003, Token: "token"},
			}
			logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
				UserRpc: stub,
			})

			mobile := "+8613800000000"
			_, err := logic.Login(&types.LoginReq{
				Mobile:   &mobile,
				Password: &tt.password,
			})
			if err == nil {
				t.Fatal("expected blank password to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastLoginReq != nil {
				t.Fatalf("blank password should not call UserRpc, got %+v", stub.lastLoginReq)
			}
		})
	}
}
