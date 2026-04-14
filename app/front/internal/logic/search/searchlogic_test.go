package search

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

type searchTestUser struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Mobile    string `gorm:"column:mobile"`
	Nickname  string `gorm:"column:nickname"`
	Avatar    string `gorm:"column:avatar"`
	Bio       string `gorm:"column:bio"`
	IsDeleted int32  `gorm:"column:is_deleted"`
}

func (searchTestUser) TableName() string {
	return "zfeed_user"
}

type searchTestContent struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	UserID      int64      `gorm:"column:user_id"`
	ContentType int32      `gorm:"column:content_type"`
	Status      int32      `gorm:"column:status"`
	Visibility  int32      `gorm:"column:visibility"`
	PublishedAt *time.Time `gorm:"column:published_at"`
	IsDeleted   int32      `gorm:"column:is_deleted"`
}

func (searchTestContent) TableName() string {
	return "zfeed_content"
}

type searchTestArticle struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	Cover       string  `gorm:"column:cover"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (searchTestArticle) TableName() string {
	return "zfeed_article"
}

type searchTestVideo struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	CoverURL    string  `gorm:"column:cover_url"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (searchTestVideo) TableName() string {
	return "zfeed_video"
}

type searchTestFollow struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64 `gorm:"column:user_id"`
	FollowUserID int64 `gorm:"column:follow_user_id"`
	Status       int32 `gorm:"column:status"`
	IsDeleted    int32 `gorm:"column:is_deleted"`
}

func (searchTestFollow) TableName() string {
	return "zfeed_follow"
}

func newSearchTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&searchTestUser{}, &searchTestContent{}, &searchTestArticle{}, &searchTestVideo{}, &searchTestFollow{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestSearchUsersReturnsFollowingState(t *testing.T) {
	db := newSearchTestDB(t)
	if err := db.Create(&[]searchTestUser{
		{ID: 1001, Mobile: "+861001", Nickname: "Alice", Avatar: "a1", Bio: "growth notes", IsDeleted: 0},
		{ID: 1002, Mobile: "+861002", Nickname: "Alicia", Avatar: "a2", Bio: "design", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&searchTestFollow{UserID: 2001, FollowUserID: 1002, Status: 10, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed follow: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(2001))
	logic := NewSearchUsersLogic(ctx, &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.SearchUsers(&types.SearchUsersReq{
		Query:    stringPtr("Ali"),
		PageSize: uint32Ptr(10),
	})
	if err != nil {
		t.Fatalf("SearchUsers returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.Items))
	}
	if !resp.Items[0].IsFollowing && !resp.Items[1].IsFollowing {
		t.Fatal("expected at least one following state to be true")
	}
}

func TestSearchContentsReturnsContentRows(t *testing.T) {
	db := newSearchTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	desc := "share growth"
	if err := db.Create(&searchTestUser{ID: 3001, Nickname: "writer", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&searchTestContent{
		ID:          4001,
		UserID:      3001,
		ContentType: 10,
		Status:      30,
		Visibility:  10,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&searchTestArticle{
		ContentID:   4001,
		Title:       "Growth Diary",
		Description: &desc,
		Cover:       "cover",
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewSearchContentsLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.SearchContents(&types.SearchContentsReq{
		Query:    stringPtr("Growth"),
		PageSize: uint32Ptr(10),
	})
	if err != nil {
		t.Fatalf("SearchContents returned error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].ContentId != 4001 {
		t.Fatalf("content_id = %d, want 4001", resp.Items[0].ContentId)
	}
}

func stringPtr(value string) *string {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}
