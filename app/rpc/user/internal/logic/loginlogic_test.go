package logic

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"zfeed/app/rpc/user/internal/common/consts/redis"
	"zfeed/app/rpc/user/internal/config"
	"zfeed/app/rpc/user/internal/model"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"
)

func TestLoginRejectsDisabledUser(t *testing.T) {
	db := newUserLogicTestDB(t)
	store, redisClient := newUserLogicTestRedisStore(t)

	const (
		mobile   = "13800002000"
		password = "secret123"
		salt     = "salt-disabled"
		userID   = 2001
	)
	passwordHash, err := utils.HashPassword(password + salt)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := db.Create(&model.ZfeedUser{
		ID:           userID,
		Username:     mobile,
		Mobile:       mobile,
		Nickname:     "disabled user",
		PasswordHash: passwordHash,
		PasswordSalt: salt,
		Status:       int32(user.UserStatus_USER_STATUS_DISABLED),
		IsDeleted:    0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
		Config:  config.Config{SessionTTL: 60},
		Redis:   redisClient,
		MysqlDb: db,
	})

	_, err = logic.Login(&user.LoginReq{
		Mobile:   mobile,
		Password: password,
	})
	if err == nil {
		t.Fatal("expected disabled user login to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusForbidden {
		t.Fatalf("error = %v, want forbidden", err)
	}
	if store.Exists(redis.BuildUserSessionUserKey(userID)) {
		t.Fatal("disabled user should not get a session")
	}
}

func TestLoginRejectsNonActiveUser(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		mobile string
		status int32
	}{
		{
			name:   "unknown status",
			userID: 2101,
			mobile: "13800002101",
			status: int32(user.UserStatus_USER_STATUS_UNKNOWN),
		},
		{
			name:   "unexpected status",
			userID: 2102,
			mobile: "13800002102",
			status: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			store, redisClient := newUserLogicTestRedisStore(t)

			const (
				password = "secret123"
				salt     = "salt-non-active"
			)
			passwordHash, err := utils.HashPassword(password + salt)
			if err != nil {
				t.Fatalf("hash password: %v", err)
			}
			if err := db.Create(&model.ZfeedUser{
				ID:           tt.userID,
				Username:     tt.mobile,
				Mobile:       tt.mobile,
				Nickname:     "non active user",
				PasswordHash: passwordHash,
				PasswordSalt: salt,
				Status:       tt.status,
				IsDeleted:    0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   redisClient,
				MysqlDb: db,
			})

			_, err = logic.Login(&user.LoginReq{
				Mobile:   tt.mobile,
				Password: password,
			})
			if err == nil {
				t.Fatal("expected non-active user login to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusForbidden {
				t.Fatalf("error = %v, want forbidden", err)
			}
			if store.Exists(redis.BuildUserSessionUserKey(tt.userID)) {
				t.Fatal("non-active user should not get a session")
			}
		})
	}
}

func TestLoginWrongPasswordHidesStatus(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		mobile string
		status int32
	}{
		{
			name:   "disabled status",
			userID: 2201,
			mobile: "13800002201",
			status: int32(user.UserStatus_USER_STATUS_DISABLED),
		},
		{
			name:   "unknown status",
			userID: 2202,
			mobile: "13800002202",
			status: int32(user.UserStatus_USER_STATUS_UNKNOWN),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			store, redisClient := newUserLogicTestRedisStore(t)

			const (
				password = "secret123"
				salt     = "salt-wrong-password"
			)
			passwordHash, err := utils.HashPassword(password + salt)
			if err != nil {
				t.Fatalf("hash password: %v", err)
			}
			if err := db.Create(&model.ZfeedUser{
				ID:           tt.userID,
				Username:     tt.mobile,
				Mobile:       tt.mobile,
				Nickname:     "hidden status user",
				PasswordHash: passwordHash,
				PasswordSalt: salt,
				Status:       tt.status,
				IsDeleted:    0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   redisClient,
				MysqlDb: db,
			})

			_, err = logic.Login(&user.LoginReq{
				Mobile:   tt.mobile,
				Password: "wrong-password",
			})
			if err == nil {
				t.Fatal("expected wrong password to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusUnauthorized {
				t.Fatalf("error = %v, want unauthorized", err)
			}
			if store.Exists(redis.BuildUserSessionUserKey(tt.userID)) {
				t.Fatal("wrong password should not get a session")
			}
		})
	}
}

func TestLoginRejectsPassword(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		mobile   string
		password string
	}{
		{
			name:     "spaces",
			userID:   2301,
			mobile:   "13800002301",
			password: "   ",
		},
		{
			name:     "tabs and newlines",
			userID:   2302,
			mobile:   "13800002302",
			password: "\t\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			store, redisClient := newUserLogicTestRedisStore(t)

			const (
				validPassword = "secret123"
				salt          = "salt-blank-password"
			)
			passwordHash, err := utils.HashPassword(validPassword + salt)
			if err != nil {
				t.Fatalf("hash password: %v", err)
			}
			if err := db.Create(&model.ZfeedUser{
				ID:           tt.userID,
				Username:     tt.mobile,
				Mobile:       tt.mobile,
				Nickname:     "blank password user",
				PasswordHash: passwordHash,
				PasswordSalt: salt,
				Status:       int32(user.UserStatus_USER_STATUS_ACTIVE),
				IsDeleted:    0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   redisClient,
				MysqlDb: db,
			})

			_, err = logic.Login(&user.LoginReq{
				Mobile:   tt.mobile,
				Password: tt.password,
			})
			if err == nil {
				t.Fatal("expected blank password to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}
			if store.Exists(redis.BuildUserSessionUserKey(tt.userID)) {
				t.Fatal("blank password should not get a session")
			}
		})
	}
}

func TestLoginHidesAvatar(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		mobile string
		avatar string
	}{
		{
			name:   "unsupported scheme",
			userID: 2401,
			mobile: "13800002401",
			avatar: "javascript:alert(1)",
		},
		{
			name:   "protocol relative",
			userID: 2402,
			mobile: "13800002402",
			avatar: "//cdn.example.com/avatar.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			_, redisClient := newUserLogicTestRedisStore(t)

			const (
				password = "secret123"
				salt     = "salt-dirty-avatar"
			)
			passwordHash, err := utils.HashPassword(password + salt)
			if err != nil {
				t.Fatalf("hash password: %v", err)
			}
			if err := db.Create(&model.ZfeedUser{
				ID:           tt.userID,
				Username:     tt.mobile,
				Mobile:       tt.mobile,
				Nickname:     "login avatar",
				Avatar:       tt.avatar,
				PasswordHash: passwordHash,
				PasswordSalt: salt,
				Status:       int32(user.UserStatus_USER_STATUS_ACTIVE),
				IsDeleted:    0,
			}).Error; err != nil {
				t.Fatalf("seed user: %v", err)
			}

			logic := NewLoginLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   redisClient,
				MysqlDb: db,
			})
			resp, err := logic.Login(&user.LoginReq{
				Mobile:   tt.mobile,
				Password: password,
			})
			if err != nil {
				t.Fatalf("Login returned error: %v", err)
			}
			if resp.GetUserId() != tt.userID || resp.GetToken() == "" {
				t.Fatalf("response = %+v, want user id and token", resp)
			}
			if resp.GetAvatar() != "" {
				t.Fatalf("avatar = %q, want empty", resp.GetAvatar())
			}
			if resp.GetNickname() != "login avatar" {
				t.Fatalf("nickname = %q, want login avatar", resp.GetNickname())
			}
		})
	}
}
