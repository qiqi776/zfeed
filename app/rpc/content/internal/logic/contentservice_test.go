package logic

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
)

type contentServiceTestContent struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	UserID        int64      `gorm:"column:user_id"`
	ContentType   int32      `gorm:"column:content_type"`
	Status        int32      `gorm:"column:status"`
	Visibility    int32      `gorm:"column:visibility"`
	LikeCount     int64      `gorm:"column:like_count"`
	FavoriteCount int64      `gorm:"column:favorite_count"`
	CommentCount  int64      `gorm:"column:comment_count"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	IsDeleted     int32      `gorm:"column:is_deleted"`
	UpdatedBy     int64      `gorm:"column:updated_by"`
}

func (contentServiceTestContent) TableName() string { return "zfeed_content" }

type contentServiceTestArticle struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	Cover       string  `gorm:"column:cover"`
	Content     string  `gorm:"column:content"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (contentServiceTestArticle) TableName() string { return "zfeed_article" }

type contentServiceTestVideo struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	OriginURL   string  `gorm:"column:origin_url"`
	CoverURL    string  `gorm:"column:cover_url"`
	Duration    int32   `gorm:"column:duration"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (contentServiceTestVideo) TableName() string { return "zfeed_video" }

type contentServiceTestUser struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Nickname  string `gorm:"column:nickname"`
	Avatar    string `gorm:"column:avatar"`
	IsDeleted int32  `gorm:"column:is_deleted"`
}

func (contentServiceTestUser) TableName() string { return "zfeed_user" }

type contentServiceTestLike struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64 `gorm:"column:user_id"`
	ContentID int64 `gorm:"column:content_id"`
	Status    int32 `gorm:"column:status"`
	IsDeleted int32 `gorm:"column:is_deleted"`
}

func (contentServiceTestLike) TableName() string { return "zfeed_like" }

type contentServiceTestFavorite struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64 `gorm:"column:user_id"`
	ContentID int64 `gorm:"column:content_id"`
	Status    int32 `gorm:"column:status"`
}

func (contentServiceTestFavorite) TableName() string { return "zfeed_favorite" }

type contentServiceTestFollow struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64 `gorm:"column:user_id"`
	FollowUserID int64 `gorm:"column:follow_user_id"`
	Status       int32 `gorm:"column:status"`
	IsDeleted    int32 `gorm:"column:is_deleted"`
}

func (contentServiceTestFollow) TableName() string { return "zfeed_follow" }

func newContentServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&contentServiceTestContent{},
		&contentServiceTestArticle{},
		&contentServiceTestVideo{},
		&contentServiceTestUser{},
		&contentServiceTestLike{},
		&contentServiceTestFavorite{},
		&contentServiceTestFollow{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestGetContentDetailAllowsAuthorToReadPrivateContent(t *testing.T) {
	db := newContentServiceTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&contentServiceTestUser{ID: 1001, Nickname: "author", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&contentServiceTestContent{
		ID:          5001,
		UserID:      1001,
		ContentType: contentTypeArticle,
		Status:      contentStatusPublish,
		Visibility:  contentVisibilityPrivate,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{
		ContentID: 5001,
		Title:     "private article",
		Cover:     "cover",
		Content:   "body",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewGetContentDetailLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.GetContentDetail(&content.GetContentDetailReq{
		ContentId: 5001,
		ViewerId:  int64Ptr(1001),
	})
	if err != nil {
		t.Fatalf("GetContentDetail returned error: %v", err)
	}
	if resp.GetDetail() == nil || resp.GetDetail().GetContentId() != 5001 || resp.GetDetail().GetTitle() != "private article" {
		t.Fatalf("unexpected detail: %+v", resp.GetDetail())
	}
}

func TestGetContentDetailRejectsPrivateContentForOtherViewer(t *testing.T) {
	db := newContentServiceTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&contentServiceTestUser{ID: 1001, Nickname: "author", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&contentServiceTestContent{
		ID:          5002,
		UserID:      1001,
		ContentType: contentTypeArticle,
		Status:      contentStatusPublish,
		Visibility:  contentVisibilityPrivate,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{
		ContentID: 5002,
		Title:     "private article",
		Cover:     "cover",
		Content:   "body",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewGetContentDetailLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	if _, err := logic.GetContentDetail(&content.GetContentDetailReq{
		ContentId: 5002,
		ViewerId:  int64Ptr(2002),
	}); err == nil {
		t.Fatal("expected private content to be hidden from other viewers")
	}
}

func TestEditArticleUpdatesSubtable(t *testing.T) {
	db := newContentServiceTestDB(t)
	if err := db.Create(&contentServiceTestContent{ID: 101, UserID: 1, ContentType: 10, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{ContentID: 101, Title: "old", Cover: "old-cover", Content: "old body", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewEditArticleLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.EditArticle(&content.EditArticleReq{
		UserId:    1,
		ContentId: 101,
		Title:     stringPtr("new-title"),
		Content:   stringPtr("new body"),
	})
	if err != nil {
		t.Fatalf("EditArticle returned error: %v", err)
	}
	if resp.GetContentId() != 101 {
		t.Fatalf("content_id = %d, want 101", resp.GetContentId())
	}

	var row contentServiceTestArticle
	if err := db.Table("zfeed_article").Where("content_id = ?", 101).Take(&row).Error; err != nil {
		t.Fatalf("query article: %v", err)
	}
	if row.Title != "new-title" || row.Content != "new body" {
		t.Fatalf("unexpected article row: %+v", row)
	}
}

func TestEditVideoUpdatesSubtable(t *testing.T) {
	db := newContentServiceTestDB(t)
	if err := db.Create(&contentServiceTestContent{ID: 202, UserID: 2, ContentType: 20, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestVideo{ContentID: 202, Title: "old", OriginURL: "old-url", CoverURL: "old-cover", Duration: 12, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed video: %v", err)
	}

	logic := NewEditVideoLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.EditVideo(&content.EditVideoReq{
		UserId:    2,
		ContentId: 202,
		Title:     stringPtr("new-video"),
		OriginUrl: stringPtr("https://example.com/new.mp4"),
		Duration:  int32Ptr(66),
	})
	if err != nil {
		t.Fatalf("EditVideo returned error: %v", err)
	}
	if resp.GetContentId() != 202 {
		t.Fatalf("content_id = %d, want 202", resp.GetContentId())
	}

	var row contentServiceTestVideo
	if err := db.Table("zfeed_video").Where("content_id = ?", 202).Take(&row).Error; err != nil {
		t.Fatalf("query video: %v", err)
	}
	if row.Title != "new-video" || row.OriginURL != "https://example.com/new.mp4" || row.Duration != 66 {
		t.Fatalf("unexpected video row: %+v", row)
	}
}

func TestDeleteContentDeletesOwnedArticle(t *testing.T) {
	db := newContentServiceTestDB(t)
	if err := db.Create(&contentServiceTestContent{
		ID:          301,
		UserID:      7,
		ContentType: 10,
		Status:      30,
		Visibility:  10,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{
		ContentID: 301,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewDeleteContentLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	if _, err := logic.DeleteContent(&content.DeleteContentReq{UserId: 7, ContentId: 301}); err != nil {
		t.Fatalf("DeleteContent returned error: %v", err)
	}

	var contentRow contentServiceTestContent
	if err := db.Table("zfeed_content").Where("id = ?", 301).Take(&contentRow).Error; err != nil {
		t.Fatalf("query deleted content: %v", err)
	}
	if contentRow.IsDeleted != 1 || contentRow.UpdatedBy != 7 {
		t.Fatalf("unexpected deleted content row: %+v", contentRow)
	}

	var articleRow contentServiceTestArticle
	if err := db.Table("zfeed_article").Where("content_id = ?", 301).Take(&articleRow).Error; err != nil {
		t.Fatalf("query deleted article: %v", err)
	}
	if articleRow.IsDeleted != 1 {
		t.Fatalf("article is_deleted = %d, want 1", articleRow.IsDeleted)
	}
}

func TestDeleteContentRejectsDeletingOthersContent(t *testing.T) {
	db := newContentServiceTestDB(t)
	if err := db.Create(&contentServiceTestContent{
		ID:          302,
		UserID:      9,
		ContentType: 20,
		Status:      30,
		Visibility:  10,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestVideo{
		ContentID: 302,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed video: %v", err)
	}

	logic := NewDeleteContentLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	if _, err := logic.DeleteContent(&content.DeleteContentReq{UserId: 7, ContentId: 302}); err == nil {
		t.Fatal("expected forbidden delete")
	}

	var contentRow contentServiceTestContent
	if err := db.Table("zfeed_content").Where("id = ?", 302).Take(&contentRow).Error; err != nil {
		t.Fatalf("query content after forbidden delete: %v", err)
	}
	if contentRow.IsDeleted != 0 {
		t.Fatalf("content is_deleted = %d, want 0", contentRow.IsDeleted)
	}
}

func int64Ptr(value int64) *int64 { return &value }

func int32Ptr(value int32) *int32 { return &value }

func stringPtr(value string) *string { return &value }
