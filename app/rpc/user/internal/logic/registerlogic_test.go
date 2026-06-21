package logic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/rpc/user/internal/config"
	"zfeed/app/rpc/user/internal/model"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
)

func TestRegisterRejectsInvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "missing domain", email: "bad-email"},
		{name: "missing suffix", email: "bad@"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   newUserLogicTestRedis(t),
				MysqlDb: db,
			})

			_, err := logic.Register(&user.RegisterReq{
				Mobile:   registerTestMobile(i),
				Password: "secret123",
				Email:    tt.email,
			})
			if err == nil {
				t.Fatal("expected invalid email to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}

			var count int64
			if err := db.Model(&model.ZfeedUser{}).Count(&count).Error; err != nil {
				t.Fatalf("count users: %v", err)
			}
			if count != 0 {
				t.Fatalf("user count = %d, want 0", count)
			}
		})
	}
}

func TestRegisterRejectsEmail(t *testing.T) {
	db := newUserLogicTestDB(t)
	store, redisClient := newUserLogicTestRedisStore(t)
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		Config:  config.Config{SessionTTL: 60},
		Redis:   redisClient,
		MysqlDb: db,
	})

	_, err := logic.Register(&user.RegisterReq{
		Mobile:   registerTestMobile(10),
		Password: "secret123",
		Email:    "Zed <zed@example.com>",
	})
	if err == nil {
		t.Fatal("expected display-name email to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("error = %v, want bad request", err)
	}

	var count int64
	if err := db.Model(&model.ZfeedUser{}).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("user count = %d, want 0", count)
	}
	if keys := store.Keys(); len(keys) != 0 {
		t.Fatalf("redis keys = %v, want empty", keys)
	}
}

func TestRegisterRejectsFutureBirthday(t *testing.T) {
	db := newUserLogicTestDB(t)
	store, redisClient := newUserLogicTestRedisStore(t)
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		Config:  config.Config{SessionTTL: 60},
		Redis:   redisClient,
		MysqlDb: db,
	})

	futureBirthday := time.Now().Add(24 * time.Hour).Unix()
	_, err := logic.Register(&user.RegisterReq{
		Mobile:   registerTestMobile(20),
		Password: "secret123",
		Birthday: futureBirthday,
	})
	if err == nil {
		t.Fatal("expected future birthday to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("error = %v, want bad request", err)
	}

	var count int64
	if err := db.Model(&model.ZfeedUser{}).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("user count = %d, want 0", count)
	}
	if keys := store.Keys(); len(keys) != 0 {
		t.Fatalf("redis keys = %v, want empty", keys)
	}
}

func TestRegisterRejectsBirthday(t *testing.T) {
	tests := []struct {
		name     string
		birthday int64
	}{
		{name: "negative one", birthday: -1},
		{name: "before unix epoch", birthday: -86400},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			store, redisClient := newUserLogicTestRedisStore(t)
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   redisClient,
				MysqlDb: db,
			})

			_, err := logic.Register(&user.RegisterReq{
				Mobile:   registerTestMobile(22 + i),
				Password: "secret123",
				Birthday: tt.birthday,
			})
			if err == nil {
				t.Fatal("expected negative birthday to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}

			var count int64
			if err := db.Model(&model.ZfeedUser{}).Count(&count).Error; err != nil {
				t.Fatalf("count users: %v", err)
			}
			if count != 0 {
				t.Fatalf("user count = %d, want 0", count)
			}
			if keys := store.Keys(); len(keys) != 0 {
				t.Fatalf("redis keys = %v, want empty", keys)
			}
		})
	}
}

func TestRegisterRejectsGender(t *testing.T) {
	db := newUserLogicTestDB(t)
	store, redisClient := newUserLogicTestRedisStore(t)
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		Config:  config.Config{SessionTTL: 60},
		Redis:   redisClient,
		MysqlDb: db,
	})
	gender := user.Gender(99)

	_, err := logic.Register(&user.RegisterReq{
		Mobile:   registerTestMobile(25),
		Password: "secret123",
		Gender:   gender,
	})
	if err == nil {
		t.Fatal("expected invalid gender to be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("error = %v, want bad request", err)
	}

	var count int64
	if err := db.Model(&model.ZfeedUser{}).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("user count = %d, want 0", count)
	}
	if keys := store.Keys(); len(keys) != 0 {
		t.Fatalf("redis keys = %v, want empty", keys)
	}
}

func TestRegisterRejectsAvatarURL(t *testing.T) {
	tests := []struct {
		name   string
		avatar string
	}{
		{name: "protocol relative", avatar: "//cdn.example.com/avatar.png"},
		{name: "unsupported scheme", avatar: "javascript:alert(1)"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			store, redisClient := newUserLogicTestRedisStore(t)
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   redisClient,
				MysqlDb: db,
			})

			_, err := logic.Register(&user.RegisterReq{
				Mobile:   registerTestMobile(40 + i),
				Password: "secret123",
				Avatar:   tt.avatar,
			})
			if err == nil {
				t.Fatal("expected invalid avatar URL to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}

			var count int64
			if err := db.Model(&model.ZfeedUser{}).Count(&count).Error; err != nil {
				t.Fatalf("count users: %v", err)
			}
			if count != 0 {
				t.Fatalf("user count = %d, want 0", count)
			}
			if keys := store.Keys(); len(keys) != 0 {
				t.Fatalf("redis keys = %v, want empty", keys)
			}
		})
	}
}

func TestRegisterTrimsBio(t *testing.T) {
	db := newUserLogicTestDB(t)
	store, redisClient := newUserLogicTestRedisStore(t)
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		Config:  config.Config{SessionTTL: 60},
		Redis:   redisClient,
		MysqlDb: db,
	})
	bio := "  clean bio  "

	resp, err := logic.Register(&user.RegisterReq{
		Mobile:   registerTestMobile(50),
		Password: "secret123",
		Bio:      &bio,
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp.GetUserId() <= 0 || resp.GetToken() == "" {
		t.Fatalf("response = %+v, want user id and token", resp)
	}

	var row model.ZfeedUser
	if err := db.First(&row, "id = ?", resp.GetUserId()).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if row.Bio != "clean bio" {
		t.Fatalf("bio = %q, want clean bio", row.Bio)
	}
	if keys := store.Keys(); len(keys) == 0 {
		t.Fatal("expected session key to be saved")
	}
}

func TestRegisterRejectsPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "spaces", password: "   "},
		{name: "tabs and newlines", password: "\t\n"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newUserLogicTestDB(t)
			store, redisClient := newUserLogicTestRedisStore(t)
			logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
				Config:  config.Config{SessionTTL: 60},
				Redis:   redisClient,
				MysqlDb: db,
			})

			_, err := logic.Register(&user.RegisterReq{
				Mobile:   registerTestMobile(60 + i),
				Password: tt.password,
			})
			if err == nil {
				t.Fatal("expected blank password to be rejected")
			}
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) || bizErr.HTTPStatus() != http.StatusBadRequest {
				t.Fatalf("error = %v, want bad request", err)
			}

			var count int64
			if err := db.Model(&model.ZfeedUser{}).Count(&count).Error; err != nil {
				t.Fatalf("count users: %v", err)
			}
			if count != 0 {
				t.Fatalf("user count = %d, want 0", count)
			}
			if keys := store.Keys(); len(keys) != 0 {
				t.Fatalf("redis keys = %v, want empty", keys)
			}
		})
	}
}

func TestRegisterRollback(t *testing.T) {
	db := newUserLogicTestDB(t)
	store, redisClient := newUserLogicTestRedisStore(t)
	store.Close()
	logic := NewRegisterLogic(context.Background(), &svc.ServiceContext{
		Config:  config.Config{SessionTTL: 60},
		Redis:   redisClient,
		MysqlDb: db,
	})

	_, err := logic.Register(&user.RegisterReq{
		Mobile:   registerTestMobile(30),
		Password: "secret123",
	})
	if err == nil {
		t.Fatal("expected register to fail when session save fails")
	}

	var count int64
	if err := db.Model(&model.ZfeedUser{}).Where("mobile = ?", "13800001030").Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("user count = %d, want 0 after session save failure", count)
	}
}

func newUserLogicTestRedis(t *testing.T) *gzredis.Redis {
	t.Helper()

	_, client := newUserLogicTestRedisStore(t)
	return client
}

func newUserLogicTestRedisStore(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	return store, client
}

func registerTestMobile(offset int) string {
	return fmt.Sprintf("138000010%02d", offset)
}
