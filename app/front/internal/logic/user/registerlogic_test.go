package user

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"
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

func TestRegisterTrimsAvatar(t *testing.T) {
	stub := &stubUserService{
		register: &userservice.RegisterRes{UserId: 3009, Token: "token"},
	}
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		UserRpc: stub,
	})

	mobile := "+8613800000000"
	password := "123456"
	avatar := "  https://example.com/avatar.png  "
	_, err := logic.Register(&types.RegisterReq{
		Mobile:   &mobile,
		Password: &password,
		Avatar:   &avatar,
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if stub.lastRegisterReq == nil {
		t.Fatal("expected register request to be forwarded")
	}
	if stub.lastRegisterReq.GetAvatar() != "https://example.com/avatar.png" {
		t.Fatalf("avatar = %q, want trimmed URL", stub.lastRegisterReq.GetAvatar())
	}
}

func TestRegisterTrimsEmail(t *testing.T) {
	stub := &stubUserService{
		register: &userservice.RegisterRes{UserId: 3010, Token: "token"},
	}
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		UserRpc: stub,
	})

	mobile := "+8613800000000"
	password := "123456"
	email := "  zed@example.com  "
	_, err := logic.Register(&types.RegisterReq{
		Mobile:   &mobile,
		Password: &password,
		Email:    &email,
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if stub.lastRegisterReq == nil {
		t.Fatal("expected register request to be forwarded")
	}
	if stub.lastRegisterReq.GetEmail() != "zed@example.com" {
		t.Fatalf("email = %q, want trimmed email", stub.lastRegisterReq.GetEmail())
	}
}

func TestRegisterNilRPC(t *testing.T) {
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		UserRpc: &stubUserService{},
	})

	mobile := "+8613800000000"
	password := "123456"
	_, err := logic.Register(&types.RegisterReq{
		Mobile:   &mobile,
		Password: &password,
	})
	if err == nil {
		t.Fatal("expected nil register rpc response to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusInternalServerError {
		t.Fatalf("error = %v, want internal server error", err)
	}
}

func TestRegisterRejectsBlankPassword(t *testing.T) {
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
				register: &userservice.RegisterRes{UserId: 3004, Token: "token"},
			}
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				UserRpc: stub,
			})

			mobile := "+8613800000000"
			_, err := logic.Register(&types.RegisterReq{
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
			if stub.lastRegisterReq != nil {
				t.Fatalf("blank password should not call UserRpc, got %+v", stub.lastRegisterReq)
			}
		})
	}
}

func TestRegisterRejectsGender(t *testing.T) {
	tests := []struct {
		name   string
		gender int32
	}{
		{name: "negative", gender: -1},
		{name: "unknown enum", gender: 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubUserService{
				register: &userservice.RegisterRes{UserId: 3005, Token: "token"},
			}
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				UserRpc: stub,
			})

			mobile := "+8613800000000"
			password := "123456"
			gender := tt.gender
			_, err := logic.Register(&types.RegisterReq{
				Mobile:   &mobile,
				Password: &password,
				Gender:   &gender,
			})
			if err == nil {
				t.Fatal("expected invalid gender to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastRegisterReq != nil {
				t.Fatalf("invalid gender should not call UserRpc, got %+v", stub.lastRegisterReq)
			}
		})
	}
}

func TestRegisterRejectsBirthday(t *testing.T) {
	tests := []struct {
		name     string
		birthday int64
	}{
		{name: "negative", birthday: -1},
		{name: "future", birthday: time.Now().Add(24 * time.Hour).Unix()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubUserService{
				register: &userservice.RegisterRes{UserId: 3006, Token: "token"},
			}
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				UserRpc: stub,
			})

			mobile := "+8613800000000"
			password := "123456"
			birthday := tt.birthday
			_, err := logic.Register(&types.RegisterReq{
				Mobile:   &mobile,
				Password: &password,
				Birthday: &birthday,
			})
			if err == nil {
				t.Fatal("expected invalid birthday to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastRegisterReq != nil {
				t.Fatalf("invalid birthday should not call UserRpc, got %+v", stub.lastRegisterReq)
			}
		})
	}
}

func TestRegisterRejectsEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "missing domain", email: "bad-email"},
		{name: "missing suffix", email: "bad@"},
		{name: "display name", email: "Zed <zed@example.com>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubUserService{
				register: &userservice.RegisterRes{UserId: 3007, Token: "token"},
			}
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				UserRpc: stub,
			})

			mobile := "+8613800000000"
			password := "123456"
			email := tt.email
			_, err := logic.Register(&types.RegisterReq{
				Mobile:   &mobile,
				Password: &password,
				Email:    &email,
			})
			if err == nil {
				t.Fatal("expected invalid email to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastRegisterReq != nil {
				t.Fatalf("invalid email should not call UserRpc, got %+v", stub.lastRegisterReq)
			}
		})
	}
}

func TestRegisterRejectsAvatar(t *testing.T) {
	tests := []struct {
		name   string
		avatar string
	}{
		{name: "protocol relative", avatar: "//cdn.example.com/avatar.png"},
		{name: "unsupported scheme", avatar: "javascript:alert(1)"},
		{name: "empty host", avatar: "https:avatar.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubUserService{
				register: &userservice.RegisterRes{UserId: 3008, Token: "token"},
			}
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				UserRpc: stub,
			})

			mobile := "+8613800000000"
			password := "123456"
			avatar := tt.avatar
			_, err := logic.Register(&types.RegisterReq{
				Mobile:   &mobile,
				Password: &password,
				Avatar:   &avatar,
			})
			if err == nil {
				t.Fatal("expected invalid avatar to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastRegisterReq != nil {
				t.Fatalf("invalid avatar should not call UserRpc, got %+v", stub.lastRegisterReq)
			}
		})
	}
}
