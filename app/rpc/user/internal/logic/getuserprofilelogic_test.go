package logic

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"zfeed/app/rpc/user/internal/model"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
)

func TestGetUserProfileRejectsNonActive(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		status int32
	}{
		{
			name:   "disabled status",
			userID: 3101,
			status: int32(user.UserStatus_USER_STATUS_DISABLED),
		},
		{
			name:   "unknown status",
			userID: 3102,
			status: int32(user.UserStatus_USER_STATUS_UNKNOWN),
		},
		{
			name:   "unexpected status",
			userID: 3103,
			status: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			if err := db.Create(&model.ZfeedUser{
				ID:        tt.userID,
				Username:  "profile-hidden",
				Mobile:    "13800003100",
				Nickname:  "profile hidden",
				Avatar:    "https://example.com/profile.png",
				Bio:       "profile bio",
				Gender:    int32(user.Gender_GENDER_MALE),
				Status:    tt.status,
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewGetUserProfileLogic(context.Background(), &svc.ServiceContext{
				MysqlDb: db,
			})
			resp, err := logic.GetUserProfile(&user.GetUserProfileReq{UserId: tt.userID})
			if err == nil {
				t.Fatalf("expected non-active profile to be hidden, got response: %+v", resp.GetUserProfile())
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusNotFound {
				t.Fatalf("error = %v, want not found", err)
			}
			if resp != nil && resp.GetUserProfile() != nil {
				t.Fatalf("user profile = %+v, want nil", resp.GetUserProfile())
			}
		})
	}
}

func TestGetUserProfileNormalizesGender(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		gender int32
	}{
		{
			name:   "too large",
			userID: 3201,
			gender: 99,
		},
		{
			name:   "negative",
			userID: 3202,
			gender: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			if err := db.Create(&model.ZfeedUser{
				ID:        tt.userID,
				Username:  "profile-gender",
				Mobile:    "13800003200",
				Nickname:  "profile gender",
				Avatar:    "https://example.com/profile-gender.png",
				Bio:       "profile bio",
				Gender:    tt.gender,
				Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewGetUserProfileLogic(context.Background(), &svc.ServiceContext{
				MysqlDb: db,
			})
			resp, err := logic.GetUserProfile(&user.GetUserProfileReq{UserId: tt.userID})
			if err != nil {
				t.Fatalf("GetUserProfile: %v", err)
			}
			if resp.GetUserProfile().GetGender() != user.Gender_GENDER_UNKNOWN {
				t.Fatalf("gender = %v, want GENDER_UNKNOWN", resp.GetUserProfile().GetGender())
			}
		})
	}
}

func TestGetUserProfileHidesAvatar(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		avatar string
	}{
		{
			name:   "unsupported scheme",
			userID: 3301,
			avatar: "javascript:alert(1)",
		},
		{
			name:   "protocol relative",
			userID: 3302,
			avatar: "//cdn.example.com/avatar.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			if err := db.Create(&model.ZfeedUser{
				ID:        tt.userID,
				Username:  "profile-avatar",
				Mobile:    "13800003300",
				Nickname:  "profile avatar",
				Avatar:    tt.avatar,
				Bio:       "profile bio",
				Gender:    int32(user.Gender_GENDER_FEMALE),
				Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewGetUserProfileLogic(context.Background(), &svc.ServiceContext{
				MysqlDb: db,
			})
			resp, err := logic.GetUserProfile(&user.GetUserProfileReq{UserId: tt.userID})
			if err != nil {
				t.Fatalf("GetUserProfile: %v", err)
			}
			if resp.GetUserProfile().GetAvatar() != "" {
				t.Fatalf("avatar = %q, want empty", resp.GetUserProfile().GetAvatar())
			}
			if resp.GetUserProfile().GetNickname() != "profile avatar" {
				t.Fatalf("nickname = %q, want profile avatar", resp.GetUserProfile().GetNickname())
			}
		})
	}
}
