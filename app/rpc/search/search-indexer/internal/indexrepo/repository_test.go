package indexrepo

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testUser struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Nickname  string `gorm:"column:nickname"`
	Avatar    string `gorm:"column:avatar"`
	Bio       string `gorm:"column:bio"`
	Mobile    string `gorm:"column:mobile"`
	Status    int32  `gorm:"column:status"`
	IsDeleted int32  `gorm:"column:is_deleted"`
}

func (testUser) TableName() string { return tableUser }

type testContent struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	UserID      int64      `gorm:"column:user_id"`
	ContentType int32      `gorm:"column:content_type"`
	Status      int32      `gorm:"column:status"`
	Visibility  int32      `gorm:"column:visibility"`
	HotScore    float64    `gorm:"column:hot_score"`
	PublishedAt *time.Time `gorm:"column:published_at"`
	IsDeleted   int32      `gorm:"column:is_deleted"`
}

func (testContent) TableName() string { return tableContent }

type testArticle struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (testArticle) TableName() string { return tableArticle }

type testVideo struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (testVideo) TableName() string { return tableVideo }

func TestRepositoryBuildContentDocument(t *testing.T) {
	db := newTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	desc := "searchable article"
	if err := db.Create(&testUser{ID: 3001, Nickname: "writer", Avatar: "avatar", Status: 10}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&testContent{
		ID:          4001,
		UserID:      3001,
		ContentType: 10,
		Status:      30,
		Visibility:  10,
		HotScore:    8.5,
		PublishedAt: &now,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&testArticle{ContentID: 4001, Title: "Growth", Description: &desc}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	doc, ok, err := New(db).BuildContentDocument(context.Background(), 4001)
	if err != nil {
		t.Fatalf("BuildContentDocument returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected document")
	}
	if doc.ContentID != 4001 || doc.Title != "Growth" || doc.Description != desc || doc.AuthorName != "writer" || doc.HotScore != 8.5 {
		t.Fatalf("unexpected content doc: %+v", doc)
	}
}

func TestRepositoryBuildContentDocumentSkipsNonPublicContent(t *testing.T) {
	db := newTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	desc := "private article"
	if err := db.Create(&testUser{ID: 3001, Nickname: "writer", Avatar: "avatar", Status: 10}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&testContent{
		ID:          4001,
		UserID:      3001,
		ContentType: 10,
		Status:      30,
		Visibility:  20,
		PublishedAt: &now,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&testArticle{ContentID: 4001, Title: "Growth", Description: &desc}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	doc, ok, err := New(db).BuildContentDocument(context.Background(), 4001)
	if err != nil {
		t.Fatalf("BuildContentDocument returned error: %v", err)
	}
	if ok || doc != nil {
		t.Fatalf("expected no document, got ok=%v doc=%+v", ok, doc)
	}
}

func TestRepositoryBuildUserDocument(t *testing.T) {
	db := newTestDB(t)
	if err := db.Create(&testUser{
		ID:        3001,
		Nickname:  "reader",
		Bio:       "bio",
		Mobile:    "+8613800000000",
		Status:    10,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	doc, ok, err := New(db).BuildUserDocument(context.Background(), 3001)
	if err != nil {
		t.Fatalf("BuildUserDocument returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected document")
	}
	if doc.UserID != 3001 || doc.Nickname != "reader" || doc.MobileSearchField != "+8613800000000" || doc.Status != 10 {
		t.Fatalf("unexpected user doc: %+v", doc)
	}
}

func TestRepositoryListContentDocumentsAfterFiltersAndOrders(t *testing.T) {
	db := newTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	desc := "searchable article"
	if err := db.Create(&testUser{ID: 3001, Nickname: "writer", Avatar: "avatar", Status: 10}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	contents := []testContent{
		{ID: 4001, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, PublishedAt: &now},
		{ID: 4002, UserID: 3001, ContentType: 10, Status: 10, Visibility: 10, PublishedAt: &now},
		{ID: 4003, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, PublishedAt: &now},
	}
	if err := db.Create(&contents).Error; err != nil {
		t.Fatalf("seed contents: %v", err)
	}
	for _, id := range []int64{4001, 4002, 4003} {
		if err := db.Create(&testArticle{ContentID: id, Title: "Growth", Description: &desc}).Error; err != nil {
			t.Fatalf("seed article: %v", err)
		}
	}

	docs, err := New(db).ListContentDocumentsAfter(context.Background(), 4001, 0, 10)
	if err != nil {
		t.Fatalf("ListContentDocumentsAfter returned error: %v", err)
	}
	if len(docs) != 1 || docs[0].ContentID != 4003 {
		t.Fatalf("docs = %+v, want only 4003", docs)
	}
}

func TestRepositoryListUserDocumentsAfterFiltersAndOrders(t *testing.T) {
	db := newTestDB(t)
	users := []testUser{
		{ID: 3001, Nickname: "old", Status: 10},
		{ID: 3002, Nickname: "deleted", Status: 10, IsDeleted: 1},
		{ID: 3003, Nickname: "active", Status: 10},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	docs, err := New(db).ListUserDocumentsAfter(context.Background(), 3001, 0, 10)
	if err != nil {
		t.Fatalf("ListUserDocumentsAfter returned error: %v", err)
	}
	if len(docs) != 1 || docs[0].UserID != 3003 {
		t.Fatalf("docs = %+v, want only 3003", docs)
	}
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&testUser{}, &testContent{}, &testArticle{}, &testVideo{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
