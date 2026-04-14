package user

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

type updateProfileTestUser struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	Mobile    string     `gorm:"column:mobile"`
	Nickname  string     `gorm:"column:nickname"`
	Avatar    string     `gorm:"column:avatar"`
	Bio       string     `gorm:"column:bio"`
	Email     string     `gorm:"column:email"`
	Gender    int32      `gorm:"column:gender"`
	Birthday  *time.Time `gorm:"column:birthday"`
	Status    int32      `gorm:"column:status"`
	IsDeleted int32      `gorm:"column:is_deleted"`
	UpdatedBy int64      `gorm:"column:updated_by"`
}

func (updateProfileTestUser) TableName() string {
	return "zfeed_user"
}

func newUpdateProfileTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&updateProfileTestUser{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestUpdateProfileUpdatesUserFields(t *testing.T) {
	db := newUpdateProfileTestDB(t)
	birthday := time.Unix(946684800, 0)
	if err := db.Create(&updateProfileTestUser{
		ID:        101,
		Mobile:    "+8613800000000",
		Nickname:  "old",
		Avatar:    "https://example.com/old.png",
		Bio:       "old bio",
		Email:     "old@example.com",
		Gender:    1,
		Birthday:  &birthday,
		Status:    10,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(101))
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
	})

	nextBirthday := time.Date(2001, 2, 3, 0, 0, 0, 0, time.UTC).Unix()
	resp, err := logic.UpdateProfile(&types.UpdateProfileReq{
		Nickname: stringPtr("new-name"),
		Avatar:   stringPtr("/uploads/avatar/new.png"),
		Bio:      stringPtr("new bio"),
		Email:    stringPtr("new@example.com"),
		Gender:   int32Ptr(2),
		Birthday: int64Ptr(nextBirthday),
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
	if resp.UserInfo.Birthday != nextBirthday {
		t.Fatalf("birthday = %d, want %d", resp.UserInfo.Birthday, nextBirthday)
	}
}

func TestUpdateProfileRejectsEmptyPayload(t *testing.T) {
	db := newUpdateProfileTestDB(t)
	ctx := context.WithValue(context.Background(), "user_id", int64(101))
	logic := NewUpdateProfileLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
	})

	if _, err := logic.UpdateProfile(&types.UpdateProfileReq{}); err == nil {
		t.Fatal("expected error for empty payload")
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
