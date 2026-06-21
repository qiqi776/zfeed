package logic

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"zfeed/app/rpc/user/internal/model"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
)

func TestUpdateProfileRejectsFutureBirthday(t *testing.T) {
	db := newUserLogicTestDB(t)
	originalBirthday := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&model.ZfeedUser{
		ID:        3301,
		Mobile:    "13800003301",
		Nickname:  "birthday user",
		Birthday:  &originalBirthday,
		Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewUpdateProfileLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})
	futureBirthday := time.Now().Add(24 * time.Hour).Unix()
	_, err := logic.UpdateProfile(&user.UpdateProfileReq{
		UserId:   3301,
		Birthday: &futureBirthday,
	})
	if err == nil {
		t.Fatal("expected future birthday to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("error = %v, want bad request", err)
	}

	var row model.ZfeedUser
	if err := db.First(&row, "id = ?", 3301).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if row.Birthday == nil || !row.Birthday.Equal(originalBirthday) {
		t.Fatalf("birthday = %v, want %v", row.Birthday, originalBirthday)
	}
}

func TestUpdateProfileRejectsNonActive(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		status int32
	}{
		{
			name:   "disabled status",
			userID: 3401,
			status: int32(user.UserStatus_USER_STATUS_DISABLED),
		},
		{
			name:   "unknown status",
			userID: 3402,
			status: int32(user.UserStatus_USER_STATUS_UNKNOWN),
		},
		{
			name:   "unexpected status",
			userID: 3403,
			status: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			if err := db.Create(&model.ZfeedUser{
				ID:        tt.userID,
				Mobile:    "13800003400",
				Nickname:  "original name",
				Status:    tt.status,
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewUpdateProfileLogic(context.Background(), &svc.ServiceContext{
				MysqlDb: db,
			})
			nickname := "updated name"
			_, err := logic.UpdateProfile(&user.UpdateProfileReq{
				UserId:   tt.userID,
				Nickname: &nickname,
			})
			if err == nil {
				t.Fatal("expected non-active user update to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusForbidden {
				t.Fatalf("error = %v, want forbidden", err)
			}

			var row model.ZfeedUser
			if err := db.First(&row, "id = ?", tt.userID).Error; err != nil {
				t.Fatalf("reload user: %v", err)
			}
			if row.Nickname != "original name" {
				t.Fatalf("nickname = %q, want original name", row.Nickname)
			}
		})
	}
}

func TestUpdateProfileRejectsAvatarURL(t *testing.T) {
	db := newUserLogicTestDB(t)
	if err := db.Create(&model.ZfeedUser{
		ID:        3501,
		Mobile:    "13800003501",
		Nickname:  "avatar user",
		Avatar:    "/avatars/original.png",
		Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewUpdateProfileLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})
	avatar := "//cdn.example.com/avatar.png"
	_, err := logic.UpdateProfile(&user.UpdateProfileReq{
		UserId: 3501,
		Avatar: &avatar,
	})
	if err == nil {
		t.Fatal("expected protocol-relative avatar URL to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("error = %v, want bad request", err)
	}

	var row model.ZfeedUser
	if err := db.First(&row, "id = ?", 3501).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if row.Avatar != "/avatars/original.png" {
		t.Fatalf("avatar = %q, want original avatar", row.Avatar)
	}
}

func TestUpdateProfileRejectsAvatarHost(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		avatar string
	}{
		{
			name:   "http empty host",
			userID: 3701,
			avatar: "http:///avatar.png",
		},
		{
			name:   "https opaque path",
			userID: 3702,
			avatar: "https:avatar.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			if err := db.Create(&model.ZfeedUser{
				ID:        tt.userID,
				Mobile:    "13800003700",
				Nickname:  "avatar host user",
				Avatar:    "/avatars/original.png",
				Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewUpdateProfileLogic(context.Background(), &svc.ServiceContext{
				MysqlDb: db,
			})
			_, err := logic.UpdateProfile(&user.UpdateProfileReq{
				UserId: tt.userID,
				Avatar: &tt.avatar,
			})
			if err == nil {
				t.Fatal("expected avatar URL without host to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}

			var row model.ZfeedUser
			if err := db.First(&row, "id = ?", tt.userID).Error; err != nil {
				t.Fatalf("reload user: %v", err)
			}
			if row.Avatar != "/avatars/original.png" {
				t.Fatalf("avatar = %q, want original avatar", row.Avatar)
			}
		})
	}
}

func TestUpdateProfileRejectsEmail(t *testing.T) {
	db := newUserLogicTestDB(t)
	if err := db.Create(&model.ZfeedUser{
		ID:        3601,
		Mobile:    "13800003601",
		Nickname:  "email user",
		Email:     "old@example.com",
		Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewUpdateProfileLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})
	email := "Zed <zed@example.com>"
	_, err := logic.UpdateProfile(&user.UpdateProfileReq{
		UserId: 3601,
		Email:  &email,
	})
	if err == nil {
		t.Fatal("expected display-name email to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("error = %v, want bad request", err)
	}

	var row model.ZfeedUser
	if err := db.First(&row, "id = ?", 3601).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if row.Email != "old@example.com" {
		t.Fatalf("email = %q, want old@example.com", row.Email)
	}
}
