package logic

import (
	"context"
	"fmt"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	followservice "zfeed/app/rpc/interaction/client/followservice"
	interactionpb "zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/search/internal/config"
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
	HotScore    float64    `gorm:"column:hot_score"`
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

func newSearchTestRedis(t *testing.T) *gzredis.Redis {
	t.Helper()

	store := miniredis.RunT(t)
	return gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
}

func newSearchSnapshotTestSvc(t *testing.T, db *gorm.DB, maxItems int) *svc.ServiceContext {
	t.Helper()

	return &svc.ServiceContext{
		Config: config.Config{
			SearchSnapshotEnabled:    true,
			SearchSnapshotTTLSeconds: 60,
			SearchSnapshotMaxItems:   maxItems,
		},
		Redis:   newSearchTestRedis(t),
		MysqlDb: db,
	}
}

func newSearchCacheTestSvc(t *testing.T, db *gorm.DB) *svc.ServiceContext {
	t.Helper()

	return &svc.ServiceContext{
		Config: config.Config{
			SearchCacheEnabled:         true,
			SearchQueryCacheTTLSeconds: 60,
			SearchDocCacheTTLSeconds:   600,
			SearchQueryCacheMaxPages:   3,
		},
		Redis:   newSearchTestRedis(t),
		MysqlDb: db,
	}
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

func TestSearchUsersQueryCacheKeepsViewerEnrichmentRealtime(t *testing.T) {
	db := newSearchTestDB(t)
	if err := db.Create(&[]searchTestUser{
		{ID: 1001, Mobile: "+861001", Nickname: "Alice", Avatar: "a1", Bio: "growth notes", IsDeleted: 0},
		{ID: 1002, Mobile: "+861002", Nickname: "Alicia", Avatar: "a2", Bio: "design", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	svcCtx := newSearchCacheTestSvc(t, db)
	svcCtx.FollowRpc = &stubSearchFollowService{
		batchQueryFollowingFunc: func(_ context.Context, in *followservice.BatchQueryFollowingReq, _ ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error) {
			if in.GetUserId() == 2002 {
				return &followservice.BatchQueryFollowingRes{
					Items: []*interactionpb.FollowingState{{UserId: 1002, IsFollowing: true}},
				}, nil
			}
			return &followservice.BatchQueryFollowingRes{
				Items: []*interactionpb.FollowingState{{UserId: 1001, IsFollowing: true}},
			}, nil
		},
	}

	logic := NewSearchUsersLogic(context.Background(), svcCtx)
	firstViewerID := int64(2001)
	firstResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Ali",
		PageSize: 10,
		ViewerId: &firstViewerID,
	})
	if err != nil {
		t.Fatalf("first SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(firstResp), []int64{1002, 1001})
	if !firstResp.Items[1].GetIsFollowing() {
		t.Fatal("expected first viewer enrichment to mark user 1001 as following")
	}

	if err := db.Create(&searchTestUser{
		ID:        2000,
		Mobile:    "+862000",
		Nickname:  "Alice New",
		Avatar:    "new",
		Bio:       "new match",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("insert concurrent user: %v", err)
	}

	secondViewerID := int64(2002)
	secondResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Ali",
		PageSize: 10,
		ViewerId: &secondViewerID,
	})
	if err != nil {
		t.Fatalf("second SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(secondResp), []int64{1002, 1001})
	if !secondResp.Items[0].GetIsFollowing() {
		t.Fatal("expected second viewer enrichment to be recomputed from follow-rpc")
	}
	if secondResp.Items[1].GetIsFollowing() {
		t.Fatal("expected cached result to avoid leaking first viewer following state")
	}

	normalized := svcCtx.NormalizeQuery("Ali")
	queryCache, err := svcCtx.Redis.GetCtx(context.Background(), searchQueryCacheKey(searchEntityUsers, searchModeLatest, normalized.Hash, 0))
	if err != nil {
		t.Fatalf("get query cache: %v", err)
	}
	if queryCache == "" {
		t.Fatal("expected query snapshot cache to be written")
	}
	docCache, err := svcCtx.Redis.GetCtx(context.Background(), searchUserDocCacheKey(1002))
	if err != nil {
		t.Fatalf("get doc cache: %v", err)
	}
	if docCache == "" {
		t.Fatal("expected user doc summary cache to be written")
	}
}

func TestSearchUsersQueryCacheStoresEmptyResult(t *testing.T) {
	db := newSearchTestDB(t)
	svcCtx := newSearchCacheTestSvc(t, db)
	logic := NewSearchUsersLogic(context.Background(), svcCtx)

	firstResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Nobody",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("first SearchUsers returned error: %v", err)
	}
	if len(firstResp.GetItems()) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(firstResp.GetItems()))
	}

	if err := db.Create(&searchTestUser{
		ID:        1001,
		Mobile:    "+861001",
		Nickname:  "Nobody",
		Avatar:    "a1",
		Bio:       "late insert",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("insert late user: %v", err)
	}

	secondResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Nobody",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("second SearchUsers returned error: %v", err)
	}
	if len(secondResp.GetItems()) != 0 {
		t.Fatalf("expected empty result cache to hide late insert during TTL, got %v", collectSearchUserIDs(secondResp))
	}
}

func TestSearchUsersQueryCacheServesConfiguredCursorWindow(t *testing.T) {
	db := newSearchTestDB(t)
	if err := db.Create(&[]searchTestUser{
		{ID: 1000, Mobile: "+861000", Nickname: "Alice 1000", Avatar: "a1", Bio: "growth", IsDeleted: 0},
		{ID: 800, Mobile: "+86800", Nickname: "Alice 800", Avatar: "a2", Bio: "growth", IsDeleted: 0},
		{ID: 600, Mobile: "+86600", Nickname: "Alice 600", Avatar: "a3", Bio: "growth", IsDeleted: 0},
		{ID: 400, Mobile: "+86400", Nickname: "Alice 400", Avatar: "a4", Bio: "growth", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	svcCtx := newSearchCacheTestSvc(t, db)
	svcCtx.Config.SearchQueryCacheMaxPages = 2
	logic := NewSearchUsersLogic(context.Background(), svcCtx)

	firstResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Alice",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("first SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(firstResp), []int64{1000, 800})

	if err := db.Create(&searchTestUser{
		ID:        700,
		Mobile:    "+86700",
		Nickname:  "Alice Late",
		Avatar:    "late",
		Bio:       "late insert",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("insert late user: %v", err)
	}

	secondResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Alice",
		PageSize: 2,
		Cursor:   firstResp.GetNextCursor(),
	})
	if err != nil {
		t.Fatalf("second SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(secondResp), []int64{600, 400})
}

func TestSearchUsersQueryCacheBypassesBeyondConfiguredCursorWindow(t *testing.T) {
	db := newSearchTestDB(t)
	if err := db.Create(&[]searchTestUser{
		{ID: 1000, Mobile: "+861000", Nickname: "Alice 1000", Avatar: "a1", Bio: "growth", IsDeleted: 0},
		{ID: 800, Mobile: "+86800", Nickname: "Alice 800", Avatar: "a2", Bio: "growth", IsDeleted: 0},
		{ID: 600, Mobile: "+86600", Nickname: "Alice 600", Avatar: "a3", Bio: "growth", IsDeleted: 0},
		{ID: 400, Mobile: "+86400", Nickname: "Alice 400", Avatar: "a4", Bio: "growth", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	svcCtx := newSearchCacheTestSvc(t, db)
	svcCtx.Config.SearchQueryCacheMaxPages = 1
	logic := NewSearchUsersLogic(context.Background(), svcCtx)

	firstResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Alice",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("first SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(firstResp), []int64{1000, 800})

	if err := db.Create(&searchTestUser{
		ID:        700,
		Mobile:    "+86700",
		Nickname:  "Alice Late",
		Avatar:    "late",
		Bio:       "late insert",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("insert late user: %v", err)
	}

	secondResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Alice",
		PageSize: 2,
		Cursor:   firstResp.GetNextCursor(),
	})
	if err != nil {
		t.Fatalf("second SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(secondResp), []int64{700, 600})
}

func TestSearchContentsQueryCacheUsesCachedCandidateWindow(t *testing.T) {
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

	svcCtx := newSearchCacheTestSvc(t, db)
	logic := NewSearchContentsLogic(context.Background(), svcCtx)
	firstResp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:    "Growth",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("first SearchContents returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchContentIDs(firstResp), []int64{4001})

	if err := db.Create(&searchTestContent{
		ID:          5000,
		UserID:      3001,
		ContentType: 10,
		Status:      30,
		Visibility:  10,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("insert late content: %v", err)
	}
	if err := db.Create(&searchTestArticle{
		ContentID:   5000,
		Title:       "Growth News",
		Description: &desc,
		Cover:       "new",
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("insert late article: %v", err)
	}

	secondResp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:    "Growth",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("second SearchContents returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchContentIDs(secondResp), []int64{4001})

	normalized := svcCtx.NormalizeQuery("Growth")
	queryCache, err := svcCtx.Redis.GetCtx(context.Background(), searchQueryCacheKey(searchEntityContents, searchModeLatest, normalized.Hash, 0))
	if err != nil {
		t.Fatalf("get content query cache: %v", err)
	}
	if queryCache == "" {
		t.Fatal("expected content query snapshot cache to be written")
	}
	docCache, err := svcCtx.Redis.GetCtx(context.Background(), searchContentDocCacheKey(4001))
	if err != nil {
		t.Fatalf("get content doc cache: %v", err)
	}
	if docCache == "" {
		t.Fatal("expected content doc summary cache to be written")
	}
}

func TestSearchContentsLatestModeOrdersByPublishedAtAndID(t *testing.T) {
	db := newSearchTestDB(t)
	older := time.Unix(1_700_000_000, 0)
	newer := older.Add(time.Hour)
	desc := "growth story"
	if err := db.Create(&searchTestUser{ID: 3001, Nickname: "writer", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&[]searchTestContent{
		{ID: 4001, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 100, PublishedAt: &older, IsDeleted: 0},
		{ID: 4002, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 1, PublishedAt: &newer, IsDeleted: 0},
		{ID: 4003, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 50, PublishedAt: &newer, IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed contents: %v", err)
	}
	for _, item := range []struct {
		id    int64
		title string
	}{
		{id: 4001, title: "Growth Older"},
		{id: 4002, title: "Growth Newer Low ID"},
		{id: 4003, title: "Growth Newer High ID"},
	} {
		if err := db.Create(&searchTestArticle{
			ContentID:   item.id,
			Title:       item.title,
			Description: &desc,
			Cover:       "cover",
			IsDeleted:   0,
		}).Error; err != nil {
			t.Fatalf("seed article: %v", err)
		}
	}

	logic := NewSearchContentsLogic(context.Background(), newSearchCacheTestSvc(t, db))
	resp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:    "Growth",
		PageSize: 10,
		Mode:     "latest",
	})
	if err != nil {
		t.Fatalf("SearchContents latest returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchContentIDs(resp), []int64{4003, 4002, 4001})
}

func TestSearchContentsRelevanceModeOrdersByTextScoreFallback(t *testing.T) {
	db := newSearchTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	query := "Growth"
	if err := db.Create(&searchTestUser{ID: 3001, Nickname: "writer", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&[]searchTestContent{
		{ID: 4001, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 0, PublishedAt: &now, IsDeleted: 0},
		{ID: 4002, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 0, PublishedAt: &now, IsDeleted: 0},
		{ID: 4003, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 0, PublishedAt: &now, IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed contents: %v", err)
	}
	descriptions := map[int64]string{
		4001: "contains Growth in body",
		4002: "other body",
		4003: "other body",
	}
	titles := map[int64]string{
		4001: "Daily Notes",
		4002: "Growth Prefix",
		4003: query,
	}
	for id, title := range titles {
		desc := descriptions[id]
		if err := db.Create(&searchTestArticle{
			ContentID:   id,
			Title:       title,
			Description: &desc,
			Cover:       "cover",
			IsDeleted:   0,
		}).Error; err != nil {
			t.Fatalf("seed article: %v", err)
		}
	}

	logic := NewSearchContentsLogic(context.Background(), newSearchSnapshotTestSvc(t, db, 10))
	resp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:    query,
		PageSize: 10,
		Mode:     "relevance",
	})
	if err != nil {
		t.Fatalf("SearchContents relevance returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchContentIDs(resp), []int64{4003, 4002, 4001})
}

func TestSearchContentsHybridModeReranksCandidateWindowByTextAndHotScore(t *testing.T) {
	db := newSearchTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	desc := "growth topic"
	if err := db.Create(&searchTestUser{ID: 3001, Nickname: "writer", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&[]searchTestContent{
		{ID: 4001, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 1, PublishedAt: &now, IsDeleted: 0},
		{ID: 4002, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 40, PublishedAt: &now, IsDeleted: 0},
		{ID: 4003, UserID: 3001, ContentType: 10, Status: 30, Visibility: 10, HotScore: 5, PublishedAt: &now, IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed contents: %v", err)
	}
	for _, item := range []struct {
		id    int64
		title string
	}{
		{id: 4001, title: "Growth Exactish"},
		{id: 4002, title: "Growth Hot"},
		{id: 4003, title: "Growth Normal"},
	} {
		if err := db.Create(&searchTestArticle{
			ContentID:   item.id,
			Title:       item.title,
			Description: &desc,
			Cover:       "cover",
			IsDeleted:   0,
		}).Error; err != nil {
			t.Fatalf("seed article: %v", err)
		}
	}

	logic := NewSearchContentsLogic(context.Background(), newSearchSnapshotTestSvc(t, db, 10))
	resp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:    "Growth",
		PageSize: 10,
		Mode:     "hybrid",
	})
	if err != nil {
		t.Fatalf("SearchContents hybrid returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchContentIDs(resp), []int64{4002, 4003, 4001})
}

func TestSearchUsersSnapshotModeReusesQueryCacheForFirstWindow(t *testing.T) {
	db := newSearchTestDB(t)
	if err := db.Create(&[]searchTestUser{
		{ID: 1001, Mobile: "+861001", Nickname: "Alice 01", Avatar: "a1", Bio: "growth", IsDeleted: 0},
		{ID: 1002, Mobile: "+861002", Nickname: "Alice 02", Avatar: "a2", Bio: "growth", IsDeleted: 0},
		{ID: 1003, Mobile: "+861003", Nickname: "Alice 03", Avatar: "a3", Bio: "growth", IsDeleted: 0},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	svcCtx := newSearchSnapshotTestSvc(t, db, 3)
	svcCtx.Config.SearchCacheEnabled = true
	svcCtx.Config.SearchQueryCacheTTLSeconds = 60
	svcCtx.Config.SearchDocCacheTTLSeconds = 600
	svcCtx.Config.SearchQueryCacheMaxPages = 3

	logic := NewSearchUsersLogic(context.Background(), svcCtx)
	firstResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Alice",
		PageSize: 2,
		Mode:     "relevance",
	})
	if err != nil {
		t.Fatalf("first relevance SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(firstResp), []int64{1003, 1002})

	if err := db.Create(&searchTestUser{
		ID:        2000,
		Mobile:    "+862000",
		Nickname:  "Alice New",
		Avatar:    "new",
		Bio:       "late insert",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("insert late user: %v", err)
	}

	secondResp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Alice",
		PageSize: 2,
		Mode:     "relevance",
	})
	if err != nil {
		t.Fatalf("second relevance SearchUsers returned error: %v", err)
	}
	assertInt64SliceEqual(t, collectSearchUserIDs(secondResp), []int64{1003, 1002})
}

func TestSearchUsersSnapshotPaginationStableAcrossInserts(t *testing.T) {
	db := newSearchTestDB(t)
	users := make([]searchTestUser, 0, 10)
	for i := 1; i <= 10; i++ {
		users = append(users, searchTestUser{
			ID:        int64(1000 + i),
			Mobile:    fmt.Sprintf("+861%03d", i),
			Nickname:  fmt.Sprintf("Alice %02d", i),
			Avatar:    fmt.Sprintf("a%d", i),
			Bio:       "snapshot user",
			IsDeleted: 0,
		})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	logic := NewSearchUsersLogic(context.Background(), newSearchSnapshotTestSvc(t, db, 10))
	resp, err := logic.SearchUsers(&searchpb.SearchUsersReq{
		Query:    "Alice",
		PageSize: 2,
		Mode:     "relevance",
	})
	if err != nil {
		t.Fatalf("first SearchUsers returned error: %v", err)
	}
	if resp.GetSnapshotId() == "" || resp.GetNextPageToken() == "" {
		t.Fatalf("expected snapshot pagination fields, got snapshot_id=%q token=%q", resp.GetSnapshotId(), resp.GetNextPageToken())
	}

	if err := db.Create(&searchTestUser{
		ID:        2000,
		Mobile:    "+862000",
		Nickname:  "Alice New",
		Avatar:    "new",
		Bio:       "snapshot user",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("insert concurrent user: %v", err)
	}

	gotIDs := collectSearchUserIDs(resp)
	pageToken := resp.GetNextPageToken()
	snapshotID := resp.GetSnapshotId()
	pageCount := 1
	for resp.GetHasMore() {
		resp, err = logic.SearchUsers(&searchpb.SearchUsersReq{
			Query:      "Alice",
			PageSize:   2,
			Mode:       "relevance",
			PageToken:  pageToken,
			SnapshotId: &snapshotID,
		})
		if err != nil {
			t.Fatalf("paged SearchUsers returned error: %v", err)
		}
		pageCount++
		gotIDs = append(gotIDs, collectSearchUserIDs(resp)...)
		pageToken = resp.GetNextPageToken()
	}

	wantIDs := []int64{1010, 1009, 1008, 1007, 1006, 1005, 1004, 1003, 1002, 1001}
	assertInt64SliceEqual(t, gotIDs, wantIDs)
	if pageCount != 5 {
		t.Fatalf("page count = %d, want 5", pageCount)
	}
}

func TestSearchContentsSnapshotPaginationStableAcrossInserts(t *testing.T) {
	db := newSearchTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&searchTestUser{ID: 3001, Nickname: "writer", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	desc := "snapshot content"
	for i := 1; i <= 3; i++ {
		contentID := int64(4000 + i)
		if err := db.Create(&searchTestContent{
			ID:          contentID,
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
			ContentID:   contentID,
			Title:       fmt.Sprintf("Growth Snapshot %d", i),
			Description: &desc,
			Cover:       fmt.Sprintf("cover%d", i),
			IsDeleted:   0,
		}).Error; err != nil {
			t.Fatalf("seed article: %v", err)
		}
	}

	logic := NewSearchContentsLogic(context.Background(), newSearchSnapshotTestSvc(t, db, 3))
	resp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:    "Growth",
		PageSize: 2,
		Mode:     "hybrid",
	})
	if err != nil {
		t.Fatalf("first SearchContents returned error: %v", err)
	}

	if err := db.Create(&searchTestContent{
		ID:          5000,
		UserID:      3001,
		ContentType: 10,
		Status:      30,
		Visibility:  10,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("insert concurrent content: %v", err)
	}
	if err := db.Create(&searchTestArticle{
		ContentID:   5000,
		Title:       "Growth Snapshot New",
		Description: &desc,
		Cover:       "new",
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("insert concurrent article: %v", err)
	}

	snapshotID := resp.GetSnapshotId()
	nextResp, err := logic.SearchContents(&searchpb.SearchContentsReq{
		Query:      "Growth",
		PageSize:   2,
		Mode:       "hybrid",
		PageToken:  resp.GetNextPageToken(),
		SnapshotId: &snapshotID,
	})
	if err != nil {
		t.Fatalf("paged SearchContents returned error: %v", err)
	}

	gotIDs := append(collectSearchContentIDs(resp), collectSearchContentIDs(nextResp)...)
	assertInt64SliceEqual(t, gotIDs, []int64{4003, 4002, 4001})
	if nextResp.GetHasMore() {
		t.Fatal("expected second snapshot content page to be the last page")
	}
}

func collectSearchUserIDs(resp *searchpb.SearchUsersRes) []int64 {
	ids := make([]int64, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		ids = append(ids, item.GetUserId())
	}
	return ids
}

func collectSearchContentIDs(resp *searchpb.SearchContentsRes) []int64 {
	ids := make([]int64, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		ids = append(ids, item.GetContentId())
	}
	return ids
}

func assertInt64SliceEqual(t *testing.T, got []int64, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %d, want %d; got=%v want=%v", i, got[i], want[i], got, want)
		}
	}
}
