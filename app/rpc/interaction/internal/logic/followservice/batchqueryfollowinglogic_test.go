package followservicelogic

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"
)

type batchFollowingTestFollow struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64 `gorm:"column:user_id"`
	FollowUserID int64 `gorm:"column:follow_user_id"`
	Status       int32 `gorm:"column:status"`
	IsDeleted    int32 `gorm:"column:is_deleted"`
}

func (batchFollowingTestFollow) TableName() string {
	return "zfeed_follow"
}

func newBatchFollowingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&batchFollowingTestFollow{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestBatchQueryFollowingReturnsRelationStates(t *testing.T) {
	db := newBatchFollowingTestDB(t)
	if err := db.Create(&[]batchFollowingTestFollow{
		{UserID: 3001, FollowUserID: 1001, Status: 10, IsDeleted: 0},
		{UserID: 3001, FollowUserID: 1003, Status: 10, IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed follows: %v", err)
	}

	resp, err := NewBatchQueryFollowingLogic(context.Background(), &svc.ServiceContext{MysqlDb: db}).BatchQueryFollowing(&interaction.BatchQueryFollowingReq{
		UserId:        3001,
		FollowUserIds: []int64{1003, 1002, 1001, 1003},
	})
	if err != nil {
		t.Fatalf("BatchQueryFollowing returned error: %v", err)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(resp.GetItems()))
	}
	if !resp.GetItems()[0].GetIsFollowing() || resp.GetItems()[1].GetIsFollowing() || !resp.GetItems()[2].GetIsFollowing() {
		t.Fatalf("unexpected items: %+v", resp.GetItems())
	}
}
