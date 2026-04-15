package logic

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	followservice "zfeed/app/rpc/interaction/client/followservice"
	interactionpb "zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/search/internal/svc"
	searchpb "zfeed/app/rpc/search/search"
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

func newSearchTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&searchTestUser{}, &searchTestContent{}, &searchTestArticle{}, &searchTestVideo{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

type stubSearchFollowService struct {
	batchQueryFollowingFunc func(ctx context.Context, in *followservice.BatchQueryFollowingReq, opts ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error)
}

func (s *stubSearchFollowService) FollowUser(context.Context, *followservice.FollowUserReq, ...grpc.CallOption) (*followservice.FollowUserRes, error) {
	return &followservice.FollowUserRes{}, nil
}

func (s *stubSearchFollowService) UnfollowUser(context.Context, *followservice.UnfollowUserReq, ...grpc.CallOption) (*followservice.UnfollowUserRes, error) {
	return &followservice.UnfollowUserRes{}, nil
}

func (s *stubSearchFollowService) ListFollowees(context.Context, *followservice.ListFolloweesReq, ...grpc.CallOption) (*followservice.ListFolloweesRes, error) {
	return &followservice.ListFolloweesRes{}, nil
}

func (s *stubSearchFollowService) ListFollowers(context.Context, *followservice.ListFollowersReq, ...grpc.CallOption) (*followservice.ListFollowersRes, error) {
	return &followservice.ListFollowersRes{}, nil
}

func (s *stubSearchFollowService) BatchQueryFollowing(ctx context.Context, in *followservice.BatchQueryFollowingReq, opts ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error) {
	return s.batchQueryFollowingFunc(ctx, in, opts...)
}

func (s *stubSearchFollowService) GetFollowSummary(context.Context, *followservice.GetFollowSummaryReq, ...grpc.CallOption) (*followservice.GetFollowSummaryRes, error) {
	return &followservice.GetFollowSummaryRes{}, nil
}

func TestSearchUsersReturnsFollowingState(t *testing.T) {
	db := newSearchTestDB(t)
	if err := db.Create(&[]searchTestUser{
		{ID: 1001, Mobile: "+861001", Nickname: "Alice", Avatar: "a1", Bio: "growth notes", IsDeleted: 0},
		{ID: 1002, Mobile: "+861002", Nickname: "Alicia", Avatar: "a2", Bio: "design", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	viewerID := int64(2001)
	logic := NewSearchUsersLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		FollowRpc: &stubSearchFollowService{
			batchQueryFollowingFunc: func(_ context.Context, in *followservice.BatchQueryFollowingReq, _ ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error) {
				if in.GetUserId() != 2001 {
					t.Fatalf("unexpected viewer_id: %+v", in)
				}
				return &followservice.BatchQueryFollowingRes{
					Items: []*interactionpb.FollowingState{
						{UserId: 1001, IsFollowing: false},
						{UserId: 1002, IsFollowing: true},
					},
				}, nil
			},
		},
	})
	resp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Ali",
		PageSize: 10,
		ViewerId: &viewerID,
	})
	if err != nil {
		t.Fatalf("SearchUsers returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.Items))
	}
	if !resp.Items[0].GetIsFollowing() && !resp.Items[1].GetIsFollowing() {
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
	resp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:    "Growth",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("SearchContents returned error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].GetContentId() != 4001 {
		t.Fatalf("content_id = %d, want 4001", resp.Items[0].GetContentId())
	}
}
