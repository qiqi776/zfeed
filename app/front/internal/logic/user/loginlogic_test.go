package user

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/client/userservice"
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
