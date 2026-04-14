package content

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/counterservice"
)

type detailTestContent struct {
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
}

func (detailTestContent) TableName() string {
	return "zfeed_content"
}

type detailTestArticle struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	Cover       string  `gorm:"column:cover"`
	Content     string  `gorm:"column:content"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (detailTestArticle) TableName() string {
	return "zfeed_article"
}

type detailTestUser struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Nickname  string `gorm:"column:nickname"`
	Avatar    string `gorm:"column:avatar"`
	IsDeleted int32  `gorm:"column:is_deleted"`
}

func (detailTestUser) TableName() string {
	return "zfeed_user"
}

type detailTestLike struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64 `gorm:"column:user_id"`
	ContentID int64 `gorm:"column:content_id"`
	Status    int32 `gorm:"column:status"`
	IsDeleted int32 `gorm:"column:is_deleted"`
}

func (detailTestLike) TableName() string {
	return "zfeed_like"
}

type detailTestFavorite struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64 `gorm:"column:user_id"`
	ContentID int64 `gorm:"column:content_id"`
	Status    int32 `gorm:"column:status"`
}

func (detailTestFavorite) TableName() string {
	return "zfeed_favorite"
}

type detailTestFollow struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64 `gorm:"column:user_id"`
	FollowUserID int64 `gorm:"column:follow_user_id"`
	Status       int32 `gorm:"column:status"`
	IsDeleted    int32 `gorm:"column:is_deleted"`
}

func (detailTestFollow) TableName() string {
	return "zfeed_follow"
}

type stubContentCounterService struct{}

func (s *stubContentCounterService) GetCount(context.Context, *counterservice.GetCountReq, ...grpc.CallOption) (*counterservice.GetCountRes, error) {
	return &counterservice.GetCountRes{}, nil
}

func (s *stubContentCounterService) BatchGetCount(_ context.Context, in *counterservice.BatchGetCountReq, _ ...grpc.CallOption) (*counterservice.BatchGetCountRes, error) {
	items := make([]*count.CountValueItem, 0, len(in.GetKeys()))
	for _, key := range in.GetKeys() {
		items = append(items, &count.CountValueItem{
			Key:   key,
			Value: 0,
		})
	}
	return &counterservice.BatchGetCountRes{Items: items}, nil
}

func (s *stubContentCounterService) Inc(context.Context, *counterservice.IncReq, ...grpc.CallOption) (*counterservice.IncRes, error) {
	return &counterservice.IncRes{}, nil
}

func (s *stubContentCounterService) Dec(context.Context, *counterservice.DecReq, ...grpc.CallOption) (*counterservice.DecRes, error) {
	return &counterservice.DecRes{}, nil
}

func (s *stubContentCounterService) GetUserProfileCounts(context.Context, *counterservice.GetUserProfileCountsReq, ...grpc.CallOption) (*counterservice.GetUserProfileCountsRes, error) {
	return &counterservice.GetUserProfileCountsRes{}, nil
}

func newContentDetailTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&detailTestContent{},
		&detailTestArticle{},
		&detailTestUser{},
		&detailTestLike{},
		&detailTestFavorite{},
		&detailTestFollow{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestGetContentDetailAllowsAuthorToReadPrivateContent(t *testing.T) {
	db := newContentDetailTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&detailTestUser{ID: 1001, Nickname: "author", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&detailTestContent{
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
	if err := db.Create(&detailTestArticle{
		ContentID: 5001,
		Title:     "private article",
		Cover:     "cover",
		Content:   "body",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(1001))
	logic := NewGetContentDetailLogic(ctx, &svc.ServiceContext{
		MysqlDb:  db,
		CountRpc: &stubContentCounterService{},
	})

	resp, err := logic.GetContentDetail(&types.GetContentDetailReq{ContentId: int64Ptr(5001)})
	if err != nil {
		t.Fatalf("GetContentDetail returned error: %v", err)
	}
	if resp.Detail.ContentId != 5001 || resp.Detail.Title != "private article" {
		t.Fatalf("unexpected detail: %+v", resp.Detail)
	}
}

func TestGetContentDetailRejectsPrivateContentForOtherViewer(t *testing.T) {
	db := newContentDetailTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&detailTestUser{ID: 1001, Nickname: "author", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&detailTestContent{
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
	if err := db.Create(&detailTestArticle{
		ContentID: 5002,
		Title:     "private article",
		Cover:     "cover",
		Content:   "body",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(2002))
	logic := NewGetContentDetailLogic(ctx, &svc.ServiceContext{
		MysqlDb:  db,
		CountRpc: &stubContentCounterService{},
	})

	if _, err := logic.GetContentDetail(&types.GetContentDetailReq{ContentId: int64Ptr(5002)}); err == nil {
		t.Fatal("expected private content to be hidden from other viewers")
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
