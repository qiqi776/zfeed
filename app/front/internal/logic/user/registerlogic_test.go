package user

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/client/userservice"
)

func TestRegisterAllowsMinimalAuthFields(t *testing.T) {
	stub := &stubUserService{
		register: &userservice.RegisterRes{
			UserId:    3001,
			Token:     "token",
			ExpiredAt: 123,
		},
	}

	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		UserRpc: stub,
	})

	mobile := "+8613800000000"
	password := "123456"
	resp, err := logic.Register(&types.RegisterReq{
		Mobile:   &mobile,
		Password: &password,
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if resp.UserId != 3001 || resp.Token != "token" || resp.ExpiredAt != 123 {
		t.Fatalf("unexpected register response: %+v", resp)
	}
	if stub.lastRegisterReq == nil {
		t.Fatal("expected register request to be forwarded")
	}
	if stub.lastRegisterReq.GetMobile() != "13800000000" || stub.lastRegisterReq.GetPassword() != password {
		t.Fatalf("unexpected forwarded register req: %+v", stub.lastRegisterReq)
	}
	if stub.lastRegisterReq.GetAvatar() != "" || stub.lastRegisterReq.GetEmail() != "" || stub.lastRegisterReq.GetBirthday() != 0 {
		t.Fatalf("expected optional profile fields to default empty, got %+v", stub.lastRegisterReq)
	}
}
