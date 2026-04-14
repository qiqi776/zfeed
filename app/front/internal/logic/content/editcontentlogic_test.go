package content

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

type editContentRow struct {
	ID          int64 `gorm:"column:id;primaryKey"`
	UserID      int64 `gorm:"column:user_id"`
	ContentType int32 `gorm:"column:content_type"`
	IsDeleted   int32 `gorm:"column:is_deleted"`
}

func (editContentRow) TableName() string {
	return "zfeed_content"
}

type editArticleRow struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	Cover       string  `gorm:"column:cover"`
	Content     string  `gorm:"column:content"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (editArticleRow) TableName() string {
	return "zfeed_article"
}

type editVideoRow struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	OriginURL   string  `gorm:"column:origin_url"`
	CoverURL    string  `gorm:"column:cover_url"`
	Duration    int32   `gorm:"column:duration"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (editVideoRow) TableName() string {
	return "zfeed_video"
}

func newEditContentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&editContentRow{}, &editArticleRow{}, &editVideoRow{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestEditArticleUpdatesSubtable(t *testing.T) {
	db := newEditContentTestDB(t)
	if err := db.Create(&editContentRow{ID: 101, UserID: 1, ContentType: 10, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&editArticleRow{ContentID: 101, Title: "old", Cover: "old-cover", Content: "old body", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(1))
	logic := NewEditArticleLogic(ctx, &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.EditArticle(&types.EditArticleReq{
		ContentId: 101,
		Title:     stringPtr("new-title"),
		Content:   stringPtr("new body"),
	})
	if err != nil {
		t.Fatalf("EditArticle returned error: %v", err)
	}
	if resp.ContentId != 101 {
		t.Fatalf("content_id = %d, want 101", resp.ContentId)
	}

	var row editArticleRow
	if err := db.Table("zfeed_article").Where("content_id = ?", 101).Take(&row).Error; err != nil {
		t.Fatalf("query article: %v", err)
	}
	if row.Title != "new-title" || row.Content != "new body" {
		t.Fatalf("unexpected article row: %+v", row)
	}
}

func TestEditVideoUpdatesSubtable(t *testing.T) {
	db := newEditContentTestDB(t)
	if err := db.Create(&editContentRow{ID: 202, UserID: 2, ContentType: 20, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&editVideoRow{ContentID: 202, Title: "old", OriginURL: "old-url", CoverURL: "old-cover", Duration: 12, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed video: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(2))
	logic := NewEditVideoLogic(ctx, &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.EditVideo(&types.EditVideoReq{
		ContentId: 202,
		Title:     stringPtr("new-video"),
		VideoUrl:  stringPtr("https://example.com/new.mp4"),
		Duration:  editInt32Ptr(66),
	})
	if err != nil {
		t.Fatalf("EditVideo returned error: %v", err)
	}
	if resp.ContentId != 202 {
		t.Fatalf("content_id = %d, want 202", resp.ContentId)
	}

	var row editVideoRow
	if err := db.Table("zfeed_video").Where("content_id = ?", 202).Take(&row).Error; err != nil {
		t.Fatalf("query video: %v", err)
	}
	if row.Title != "new-video" || row.OriginURL != "https://example.com/new.mp4" || row.Duration != 66 {
		t.Fatalf("unexpected video row: %+v", row)
	}
}

func stringPtr(value string) *string {
	return &value
}

func editInt32Ptr(value int32) *int32 {
	return &value
}
