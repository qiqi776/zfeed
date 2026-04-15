package followservicelogic

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"
)

type listFollowersTestFollow struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64 `gorm:"column:user_id"`
	FollowUserID int64 `gorm:"column:follow_user_id"`
	Status       int32 `gorm:"column:status"`
	IsDeleted    int32 `gorm:"column:is_deleted"`
}

func (listFollowersTestFollow) TableName() string {
	return "zfeed_follow"
}

type listFollowersTestUser struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Nickname  string `gorm:"column:nickname"`
	Avatar    string `gorm:"column:avatar"`
	Bio       string `gorm:"column:bio"`
	IsDeleted int32  `gorm:"column:is_deleted"`
}

func (listFollowersTestUser) TableName() string {
	return "zfeed_user"
}

func newListFollowersTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&listFollowersTestFollow{}, &listFollowersTestUser{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestListFollowersReturnsProfilesAndFollowingState(t *testing.T) {
	db := newListFollowersTestDB(t)

	if err := db.Create(&[]listFollowersTestUser{
		{ID: 1001, Nickname: "Alice", Avatar: "a1", Bio: "growth", IsDeleted: 0},
		{ID: 1002, Nickname: "Alicia", Avatar: "a2", Bio: "design", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]listFollowersTestFollow{
		{UserID: 1001, FollowUserID: 9001, Status: 10, IsDeleted: 0},
		{UserID: 1002, FollowUserID: 9001, Status: 10, IsDeleted: 0},
		{UserID: 3001, FollowUserID: 1002, Status: 10, IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed follows: %v", err)
	}

	viewerID := int64(3001)
	logic := NewListFollowersLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.ListFollowers(&interaction.ListFollowersReq{
		UserId:   9001,
		PageSize: 10,
		ViewerId: &viewerID,
	})
	if err != nil {
		t.Fatalf("ListFollowers returned error: %v", err)
	}
	if len(resp.GetItems()) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.GetItems()))
	}
	if resp.GetItems()[1].GetUserId() != 1001 || !resp.GetItems()[0].GetIsFollowing() {
		t.Fatalf("unexpected items: %+v", resp.GetItems())
	}
}
