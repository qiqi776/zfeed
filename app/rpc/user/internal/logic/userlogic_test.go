package logic

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/user/internal/model"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
)

func newUserLogicTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedUser{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestGetMeReturnsPrivateProfileFields(t *testing.T) {
	db := newUserLogicTestDB(t)
	birthday := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&model.ZfeedUser{
		ID:        1001,
		Username:  "alice",
		Mobile:    "13800000000",
		Nickname:  "Alice",
		Avatar:    "https://example.com/a.png",
		Bio:       "hello",
		Email:     "alice@example.com",
		Gender:    2,
		Birthday:  &birthday,
		Status:    10,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewGetMeLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})
	resp, err := logic.GetMe(&user.GetMeReq{UserId: 1001})
	if err != nil {
		t.Fatalf("GetMe returned error: %v", err)
	}
	if resp.GetUserInfo() == nil {
		t.Fatal("expected user info")
	}
	if resp.GetUserInfo().GetEmail() != "alice@example.com" {
		t.Fatalf("email = %q, want %q", resp.GetUserInfo().GetEmail(), "alice@example.com")
	}
	if resp.GetUserInfo().GetBirthday() != birthday.Unix() {
		t.Fatalf("birthday = %d, want %d", resp.GetUserInfo().GetBirthday(), birthday.Unix())
	}
}

func TestUpdateProfilePersistsUserFields(t *testing.T) {
	db := newUserLogicTestDB(t)
	oldBirthday := time.Date(1999, 1, 2, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&model.ZfeedUser{
		ID:        1002,
		Mobile:    "13800000001",
		Nickname:  "old",
		Avatar:    "https://example.com/old.png",
		Bio:       "old bio",
		Email:     "old@example.com",
		Gender:    1,
		Birthday:  &oldBirthday,
		Status:    10,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewUpdateProfileLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})

	nickname := "new-name"
	avatar := "/uploads/avatar/new.png"
	bio := "new bio"
	email := "new@example.com"
	gender := user.Gender_GENDER_FEMALE
	birthday := time.Date(2001, 2, 3, 0, 0, 0, 0, time.UTC).Unix()
	resp, err := logic.UpdateProfile(&user.UpdateProfileReq{
		UserId:   1002,
		Nickname: &nickname,
		Avatar:   &avatar,
		Bio:      &bio,
		Email:    &email,
		Gender:   &gender,
		Birthday: &birthday,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if resp.GetUserInfo() == nil {
		t.Fatal("expected user info")
	}
	if resp.GetUserInfo().GetNickname() != nickname {
		t.Fatalf("nickname = %q, want %q", resp.GetUserInfo().GetNickname(), nickname)
	}
	if resp.GetUserInfo().GetEmail() != email {
		t.Fatalf("email = %q, want %q", resp.GetUserInfo().GetEmail(), email)
	}
	if resp.GetUserInfo().GetBirthday() != birthday {
		t.Fatalf("birthday = %d, want %d", resp.GetUserInfo().GetBirthday(), birthday)
	}

	var row model.ZfeedUser
	if err := db.First(&row, "id = ?", 1002).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if row.Nickname != nickname || row.Email != email || row.Avatar != avatar || row.Bio != bio {
		t.Fatalf("unexpected persisted row: %+v", row)
	}
	if row.Birthday == nil || row.Birthday.Unix() != birthday {
		t.Fatalf("persisted birthday = %v, want %d", row.Birthday, birthday)
	}
	if row.UpdatedBy != 1002 {
		t.Fatalf("updated_by = %d, want 1002", row.UpdatedBy)
	}
}
