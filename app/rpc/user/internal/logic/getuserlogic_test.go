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

func TestGetUserRejectsNonActive(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		status int32
	}{
		{
			name:   "disabled status",
			userID: 3001,
			status: int32(user.UserStatus_USER_STATUS_DISABLED),
		},
		{
			name:   "unknown status",
			userID: 3002,
			status: int32(user.UserStatus_USER_STATUS_UNKNOWN),
		},
		{
			name:   "unexpected status",
			userID: 3003,
			status: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			if err := db.Create(&model.ZfeedUser{
				ID:        tt.userID,
				Username:  "hidden-user",
				Mobile:    "13800003000",
				Nickname:  "hidden user",
				Avatar:    "https://example.com/hidden.png",
				Bio:       "hidden bio",
				Status:    tt.status,
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewGetUserLogic(context.Background(), &svc.ServiceContext{
				MysqlDb: db,
			})
			resp, err := logic.GetUser(&user.GetUserReq{UserId: tt.userID})
			if err == nil {
				t.Fatalf("expected non-active user to be hidden, got response: %+v", resp.GetUserInfo())
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusNotFound {
				t.Fatalf("error = %v, want not found", err)
			}
			if resp != nil && resp.GetUserInfo() != nil {
				t.Fatalf("user info = %+v, want nil", resp.GetUserInfo())
			}
		})
	}
}

func TestGetUserHidesMobile(t *testing.T) {
	db := newUserLogicTestDB(t)
	if err := db.Create(&model.ZfeedUser{
		ID:        3701,
		Username:  "public-user",
		Mobile:    "13800003701",
		Nickname:  "public name",
		Avatar:    "https://example.com/public.png",
		Bio:       "public bio",
		Gender:    int32(user.Gender_GENDER_FEMALE),
		Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewGetUserLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})
	resp, err := logic.GetUser(&user.GetUserReq{UserId: 3701})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	info := resp.GetUserInfo()
	if info.GetUserId() != 3701 || info.GetNickname() != "public name" {
		t.Fatalf("user info = %+v, want public user fields", info)
	}
	if info.GetMobile() != "" {
		t.Fatalf("mobile = %q, want empty", info.GetMobile())
	}
}
