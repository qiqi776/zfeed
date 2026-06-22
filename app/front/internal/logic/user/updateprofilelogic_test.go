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
	userpb "zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
)

func TestUpdateProfileUpdatesUserFields(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(101))
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{
			update: &userservice.UpdateProfileRes{
				UserInfo: &userservice.UserInfo{
					UserId:   101,
					Mobile:   "+8613800000000",
					Nickname: "new-name",
					Avatar:   "/uploads/avatar/new.png",
					Bio:      "new bio",
					Email:    "new@example.com",
					Gender:   userpb.Gender_GENDER_FEMALE,
					Birthday: 981158400,
					Status:   userpb.UserStatus_USER_STATUS_ACTIVE,
				},
			},
		},
	})

	resp, err := logic.UpdateProfile(&types.UpdateProfileReq{
		Nickname: stringPtr("new-name"),
		Avatar:   stringPtr("/uploads/avatar/new.png"),
		Bio:      stringPtr("new bio"),
		Email:    stringPtr("new@example.com"),
		Gender:   int32Ptr(2),
		Birthday: int64Ptr(981158400),
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if resp.UserInfo.Nickname != "new-name" {
		t.Fatalf("nickname = %q, want %q", resp.UserInfo.Nickname, "new-name")
	}
	if resp.UserInfo.Avatar != "/uploads/avatar/new.png" {
		t.Fatalf("avatar = %q, want %q", resp.UserInfo.Avatar, "/uploads/avatar/new.png")
	}
	if resp.UserInfo.Bio != "new bio" {
		t.Fatalf("bio = %q, want %q", resp.UserInfo.Bio, "new bio")
	}
	if resp.UserInfo.Gender != 2 {
		t.Fatalf("gender = %d, want 2", resp.UserInfo.Gender)
	}
	if resp.UserInfo.Email != "new@example.com" {
		t.Fatalf("email = %q, want %q", resp.UserInfo.Email, "new@example.com")
	}
	if resp.UserInfo.Birthday != 981158400 {
		t.Fatalf("birthday = %d, want %d", resp.UserInfo.Birthday, 981158400)
	}
}

func TestUpdateProfileRejectsEmptyPayload(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(101))
	stub := &stubUserService{}
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{UserRpc: stub})

	_, err := logic.UpdateProfile(&types.UpdateProfileReq{})
	if err == nil {
		t.Fatal("expected error for empty payload")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("error = %v, want bad request", err)
	}
	if stub.lastUpdateReq != nil {
		t.Fatalf("empty payload should not call UserRpc, got %+v", stub.lastUpdateReq)
	}
}

func TestUpdateProfileRejectsGender(t *testing.T) {
	tests := []struct {
		name   string
		gender int32
	}{
		{name: "negative", gender: -1},
		{name: "unknown enum", gender: 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "user_id", int64(101))
			stub := &stubUserService{
				update: &userservice.UpdateProfileRes{
					UserInfo: &userservice.UserInfo{UserId: 101},
				},
			}
			logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{UserRpc: stub})

			_, err := logic.UpdateProfile(&types.UpdateProfileReq{
				Gender: int32Ptr(tt.gender),
			})
			if err == nil {
				t.Fatal("expected invalid gender to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastUpdateReq != nil {
				t.Fatalf("invalid gender should not call UserRpc, got %+v", stub.lastUpdateReq)
			}
		})
	}
}

func TestUpdateProfileRejectsEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "empty", email: ""},
		{name: "spaces", email: "   "},
		{name: "malformed", email: "bad-email"},
		{name: "missing suffix", email: "bad@"},
		{name: "display name", email: "Zed <zed@example.com>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "user_id", int64(101))
			stub := &stubUserService{
				update: &userservice.UpdateProfileRes{
					UserInfo: &userservice.UserInfo{UserId: 101},
				},
			}
			logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{UserRpc: stub})

			_, err := logic.UpdateProfile(&types.UpdateProfileReq{
				Email: &tt.email,
			})
			if err == nil {
				t.Fatal("expected invalid email to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastUpdateReq != nil {
				t.Fatalf("invalid email should not call UserRpc, got %+v", stub.lastUpdateReq)
			}
		})
	}
}

func TestUpdateProfileRejectsAvatar(t *testing.T) {
	tests := []struct {
		name   string
		avatar string
	}{
		{name: "empty", avatar: ""},
		{name: "spaces", avatar: "   "},
		{name: "protocol relative", avatar: "//cdn.example.com/avatar.png"},
		{name: "javascript", avatar: "javascript:alert(1)"},
		{name: "missing host", avatar: "https:avatar.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "user_id", int64(101))
			stub := &stubUserService{
				update: &userservice.UpdateProfileRes{
					UserInfo: &userservice.UserInfo{UserId: 101},
				},
			}
			logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{UserRpc: stub})

			_, err := logic.UpdateProfile(&types.UpdateProfileReq{
				Avatar: &tt.avatar,
			})
			if err == nil {
				t.Fatal("expected invalid avatar to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastUpdateReq != nil {
				t.Fatalf("invalid avatar should not call UserRpc, got %+v", stub.lastUpdateReq)
			}
		})
	}
}

func TestUpdateProfileTrimsAvatar(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(101))
	stub := &stubUserService{
		update: &userservice.UpdateProfileRes{
			UserInfo: &userservice.UserInfo{
				UserId: 101,
				Avatar: "https://example.com/avatar.png",
			},
		},
	}
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{UserRpc: stub})

	avatar := "  https://example.com/avatar.png  "
	if _, err := logic.UpdateProfile(&types.UpdateProfileReq{
		Avatar: &avatar,
	}); err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if stub.lastUpdateReq == nil || stub.lastUpdateReq.Avatar == nil {
		t.Fatal("expected avatar to be forwarded")
	}
	if got := stub.lastUpdateReq.GetAvatar(); got != "https://example.com/avatar.png" {
		t.Fatalf("avatar = %q, want %q", got, "https://example.com/avatar.png")
	}
}

func TestUpdateProfileTrimsEmail(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(101))
	stub := &stubUserService{
		update: &userservice.UpdateProfileRes{
			UserInfo: &userservice.UserInfo{
				UserId: 101,
				Email:  "zed@example.com",
			},
		},
	}
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{UserRpc: stub})

	email := "  zed@example.com  "
	if _, err := logic.UpdateProfile(&types.UpdateProfileReq{
		Email: &email,
	}); err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if stub.lastUpdateReq == nil || stub.lastUpdateReq.Email == nil {
		t.Fatal("expected email to be forwarded")
	}
	if got := stub.lastUpdateReq.GetEmail(); got != "zed@example.com" {
		t.Fatalf("email = %q, want %q", got, "zed@example.com")
	}
}

func TestUpdateProfileRejectsBirthday(t *testing.T) {
	tests := []struct {
		name     string
		birthday int64
	}{
		{name: "zero", birthday: 0},
		{name: "negative", birthday: -1},
		{name: "future", birthday: time.Now().Add(24 * time.Hour).Unix()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "user_id", int64(101))
			stub := &stubUserService{
				update: &userservice.UpdateProfileRes{
					UserInfo: &userservice.UserInfo{UserId: 101},
				},
			}
			logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{UserRpc: stub})

			_, err := logic.UpdateProfile(&types.UpdateProfileReq{
				Birthday: &tt.birthday,
			})
			if err == nil {
				t.Fatal("expected invalid birthday to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if stub.lastUpdateReq != nil {
				t.Fatalf("invalid birthday should not call UserRpc, got %+v", stub.lastUpdateReq)
			}
		})
	}
}

func TestUpdateProfileFailsWhenUserRPCFails(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(101))
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{err: errors.New("rpc failed")},
	})

	if _, err := logic.UpdateProfile(&types.UpdateProfileReq{
		Nickname: stringPtr("new-name"),
	}); err == nil {
		t.Fatal("expected user rpc failure")
	}
}

func stringPtr(value string) *string {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
