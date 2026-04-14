package content

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

type testContentRow struct {
	ID          int64 `gorm:"column:id;primaryKey"`
	UserID      int64 `gorm:"column:user_id"`
	ContentType int32 `gorm:"column:content_type"`
	Status      int32 `gorm:"column:status"`
	Visibility  int32 `gorm:"column:visibility"`
	IsDeleted   int32 `gorm:"column:is_deleted"`
	UpdatedBy   int64 `gorm:"column:updated_by"`
}

func (testContentRow) TableName() string {
	return "zfeed_content"
}

type testArticleRow struct {
	ID        int64 `gorm:"column:id;primaryKey"`
	ContentID int64 `gorm:"column:content_id"`
	IsDeleted int32 `gorm:"column:is_deleted"`
}

func (testArticleRow) TableName() string {
	return "zfeed_article"
}

type testVideoRow struct {
	ID        int64 `gorm:"column:id;primaryKey"`
	ContentID int64 `gorm:"column:content_id"`
	IsDeleted int32 `gorm:"column:is_deleted"`
}

func (testVideoRow) TableName() string {
	return "zfeed_video"
}

func openDeleteContentTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&testContentRow{}, &testArticleRow{}, &testVideoRow{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func TestDeleteContentLogic_DeletesOwnedArticle(t *testing.T) {
	db := openDeleteContentTestDB(t, "file:front_delete_content_owned?mode=memory&cache=shared")
	if err := db.Create(&testContentRow{
		ID:          101,
		UserID:      7,
		ContentType: 10,
		Status:      30,
		Visibility:  10,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&testArticleRow{
		ID:        1,
		ContentID: 101,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(7))
	logic := NewDeleteContentLogic(ctx, &svc.ServiceContext{MysqlDb: db})

	if _, err := logic.DeleteContent(&types.DeleteContentReq{ContentId: 101}); err != nil {
		t.Fatalf("DeleteContent returned error: %v", err)
	}

	var contentRow testContentRow
	if err := db.Table("zfeed_content").Where("id = ?", 101).Take(&contentRow).Error; err != nil {
		t.Fatalf("query deleted content: %v", err)
	}
	if contentRow.IsDeleted != 1 {
		t.Fatalf("content is_deleted = %d, want 1", contentRow.IsDeleted)
	}
	if contentRow.UpdatedBy != 7 {
		t.Fatalf("content updated_by = %d, want 7", contentRow.UpdatedBy)
	}

	var articleRow testArticleRow
	if err := db.Table("zfeed_article").Where("content_id = ?", 101).Take(&articleRow).Error; err != nil {
		t.Fatalf("query deleted article: %v", err)
	}
	if articleRow.IsDeleted != 1 {
		t.Fatalf("article is_deleted = %d, want 1", articleRow.IsDeleted)
	}
}

func TestDeleteContentLogic_RejectsDeletingOthersContent(t *testing.T) {
	db := openDeleteContentTestDB(t, "file:front_delete_content_forbidden?mode=memory&cache=shared")
	if err := db.Create(&testContentRow{
		ID:          202,
		UserID:      9,
		ContentType: 20,
		Status:      30,
		Visibility:  10,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&testVideoRow{
		ID:        1,
		ContentID: 202,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed video: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(7))
	logic := NewDeleteContentLogic(ctx, &svc.ServiceContext{MysqlDb: db})

	if _, err := logic.DeleteContent(&types.DeleteContentReq{ContentId: 202}); err == nil {
		t.Fatal("DeleteContent should reject deleting others content")
	}

	var contentRow testContentRow
	if err := db.Table("zfeed_content").Where("id = ?", 202).Take(&contentRow).Error; err != nil {
		t.Fatalf("query content after forbidden delete: %v", err)
	}
	if contentRow.IsDeleted != 0 {
		t.Fatalf("content is_deleted = %d, want 0", contentRow.IsDeleted)
	}
}
