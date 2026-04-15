package user

import (
	"context"
	"errors"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/client/userservice"
	userpb "zfeed/app/rpc/user/user"
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
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{},
	})

	if _, err := logic.UpdateProfile(&types.UpdateProfileReq{}); err == nil {
		t.Fatal("expected error for empty payload")
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
