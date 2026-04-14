package user

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

type followerTestUser struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Nickname  string `gorm:"column:nickname"`
	Avatar    string `gorm:"column:avatar"`
	Bio       string `gorm:"column:bio"`
	IsDeleted int32  `gorm:"column:is_deleted"`
}

func (followerTestUser) TableName() string {
	return "zfeed_user"
}

type followerTestRelation struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64 `gorm:"column:user_id"`
	FollowUserID int64 `gorm:"column:follow_user_id"`
	Status       int32 `gorm:"column:status"`
	IsDeleted    int32 `gorm:"column:is_deleted"`
}

func (followerTestRelation) TableName() string {
	return "zfeed_follow"
}

func newQueryFollowersTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&followerTestUser{}, &followerTestRelation{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestQueryFollowersReturnsOrderedUsers(t *testing.T) {
	db := newQueryFollowersTestDB(t)

	if err := db.Create(&[]followerTestUser{
		{ID: 1001, Nickname: "u1001", Avatar: "a1", Bio: "b1", IsDeleted: 0},
		{ID: 1002, Nickname: "u1002", Avatar: "a2", Bio: "b2", IsDeleted: 0},
		{ID: 1003, Nickname: "u1003", Avatar: "a3", Bio: "b3", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	if err := db.Create(&[]followerTestRelation{
		{UserID: 1001, FollowUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1002, FollowUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1003, FollowUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 3001, FollowUserID: 1002, Status: 10, IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed relations: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(3001))
	logic := NewQueryFollowersLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
	})

	resp, err := logic.QueryFollowers(&types.QueryFollowersReq{
		UserId:   int64Ptr(2001),
		PageSize: uint32Ptr(2),
	})
	if err != nil {
		t.Fatalf("QueryFollowers returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.Items))
	}
	if resp.Items[0].UserId != 1003 || resp.Items[1].UserId != 1002 {
		t.Fatalf("user ids = [%d %d], want [1003 1002]", resp.Items[0].UserId, resp.Items[1].UserId)
	}
	if !resp.Items[1].IsFollowing {
		t.Fatal("viewer following state for 1002 = false, want true")
	}
	if !resp.HasMore || resp.NextCursor != 1002 {
		t.Fatalf("pagination = has_more:%v next_cursor:%d, want true/1002", resp.HasMore, resp.NextCursor)
	}
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}
