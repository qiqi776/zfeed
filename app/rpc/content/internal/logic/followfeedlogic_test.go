package logic

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/svc"
	followservice "zfeed/app/rpc/interaction/client/followservice"
)

var _ followservice.FollowService = (*fakeFollowService)(nil)

type fakeFollowService struct {
	listFolloweesFunc func(ctx context.Context, in *followservice.ListFolloweesReq, opts ...grpc.CallOption) (*followservice.ListFolloweesRes, error)
}

func (f *fakeFollowService) FollowUser(context.Context, *followservice.FollowUserReq, ...grpc.CallOption) (*followservice.FollowUserRes, error) {
	return nil, errors.New("unexpected FollowUser call")
}

func (f *fakeFollowService) UnfollowUser(context.Context, *followservice.UnfollowUserReq, ...grpc.CallOption) (*followservice.UnfollowUserRes, error) {
	return nil, errors.New("unexpected UnfollowUser call")
}

func (f *fakeFollowService) ListFollowees(ctx context.Context, in *followservice.ListFolloweesReq, opts ...grpc.CallOption) (*followservice.ListFolloweesRes, error) {
	if f.listFolloweesFunc == nil {
		return nil, errors.New("unexpected ListFollowees call")
	}
	return f.listFolloweesFunc(ctx, in, opts...)
}

func (f *fakeFollowService) ListFollowers(context.Context, *followservice.ListFollowersReq, ...grpc.CallOption) (*followservice.ListFollowersRes, error) {
	return nil, errors.New("unexpected ListFollowers call")
}

func (f *fakeFollowService) BatchQueryFollowing(context.Context, *followservice.BatchQueryFollowingReq, ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error) {
	return nil, errors.New("unexpected BatchQueryFollowing call")
}

func (f *fakeFollowService) GetFollowSummary(context.Context, *followservice.GetFollowSummaryReq, ...grpc.CallOption) (*followservice.GetFollowSummaryRes, error) {
	return nil, errors.New("unexpected GetFollowSummary call")
}

type followFeedSeed struct {
	contentID   int64
	authorID    int64
	contentType contentpb.ContentType
	title       string
	coverURL    string
}

func newFollowFeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedContent{}, &model.ZfeedArticle{}, &model.ZfeedVideo{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedFollowFeedRows(t *testing.T, db *gorm.DB, rows []followFeedSeed) {
	t.Helper()

	for _, row := range rows {
		publishedAt := time.Unix(row.contentID, 0)
		contentRow := &model.ZfeedContent{
			ID:          row.contentID,
			UserID:      row.authorID,
			ContentType: int32(row.contentType),
			Status:      int32(contentpb.ContentStatus_CONTENT_STATUS_PUBLISHED),
			Visibility:  int32(contentpb.Visibility_VISIBILITY_PUBLIC),
			PublishedAt: &publishedAt,
			IsDeleted:   0,
		}
		if err := db.Create(contentRow).Error; err != nil {
			t.Fatalf("create content row %d: %v", row.contentID, err)
		}

		switch row.contentType {
		case contentpb.ContentType_CONTENT_TYPE_ARTICLE:
			if err := db.Create(&model.ZfeedArticle{
				ContentID: row.contentID,
				Title:     row.title,
				Cover:     row.coverURL,
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("create article row %d: %v", row.contentID, err)
			}
		case contentpb.ContentType_CONTENT_TYPE_VIDEO:
			if err := db.Create(&model.ZfeedVideo{
				ContentID: row.contentID,
				Title:     row.title,
				CoverURL:  row.coverURL,
				IsDeleted: 0,
			}).Error; err != nil {
				t.Fatalf("create video row %d: %v", row.contentID, err)
			}
		}
	}
}

func newFollowFeedRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	return store, client
}

func TestFollowFeed_InboxHitReturnsOrderedItems(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 3003, authorID: 2001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-3003", coverURL: "cover-a-3003"},
		{contentID: 3002, authorID: 2002, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-3002", coverURL: "cover-v-3002"},
		{contentID: 3001, authorID: 2001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-3001", coverURL: "cover-a-3001"},
	})

	inboxKey := redisconsts.BuildFollowInboxKey(1001)
	for _, contentID := range []int64{3003, 3002, 3001} {
		store.ZAdd(inboxKey, float64(contentID), strconv.FormatInt(contentID, 10))
	}

	logic := NewFollowFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.FollowFeed(&contentpb.FollowFeedReq{
		UserId:   1001,
		Cursor:   "",
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("FollowFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(resp.GetItems()))
	}
	if resp.GetItems()[0].GetContentId() != 3003 || resp.GetItems()[1].GetContentId() != 3002 || resp.GetItems()[2].GetContentId() != 3001 {
		t.Fatalf("content ids = [%d %d %d], want [3003 3002 3001]",
			resp.GetItems()[0].GetContentId(), resp.GetItems()[1].GetContentId(), resp.GetItems()[2].GetContentId())
	}
	if resp.GetItems()[0].GetTitle() != "article-3003" || resp.GetItems()[1].GetTitle() != "video-3002" {
		t.Fatalf("titles = [%s %s], want [article-3003 video-3002]", resp.GetItems()[0].GetTitle(), resp.GetItems()[1].GetTitle())
	}
	if resp.GetHasMore() {
		t.Fatal("has_more = true, want false")
	}
}

func TestFollowFeed_CursorPagination(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 3103, authorID: 2001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-3103", coverURL: "cover-a-3103"},
		{contentID: 3102, authorID: 2002, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-3102", coverURL: "cover-v-3102"},
		{contentID: 3101, authorID: 2003, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-3101", coverURL: "cover-a-3101"},
	})

	inboxKey := redisconsts.BuildFollowInboxKey(1002)
	for _, contentID := range []int64{3103, 3102, 3101} {
		store.ZAdd(inboxKey, float64(contentID), strconv.FormatInt(contentID, 10))
	}

	logic := NewFollowFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	firstPage, err := logic.FollowFeed(&contentpb.FollowFeedReq{
		UserId:   1002,
		Cursor:   "",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("first page returned error: %v", err)
	}
	if len(firstPage.GetItems()) != 2 {
		t.Fatalf("len(firstPage.items) = %d, want 2", len(firstPage.GetItems()))
	}
	if !firstPage.GetHasMore() || firstPage.GetNextCursor() != "3102" {
		t.Fatalf("first page pagination = has_more:%v next_cursor:%s, want true/3102", firstPage.GetHasMore(), firstPage.GetNextCursor())
	}

	secondPage, err := logic.FollowFeed(&contentpb.FollowFeedReq{
		UserId:   1002,
		Cursor:   firstPage.GetNextCursor(),
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("second page returned error: %v", err)
	}
	if len(secondPage.GetItems()) != 1 || secondPage.GetItems()[0].GetContentId() != 3101 {
		t.Fatalf("second page items = %+v, want [3101]", secondPage.GetItems())
	}
	if secondPage.GetHasMore() || secondPage.GetNextCursor() != "" {
		t.Fatalf("second page pagination = has_more:%v next_cursor:%s, want false/empty", secondPage.GetHasMore(), secondPage.GetNextCursor())
	}
}

func TestFollowFeed_MissRebuildsInboxAndSubsequentReadHitsCache(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 4103, authorID: 2201, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-4103", coverURL: "cover-a-4103"},
		{contentID: 4102, authorID: 2202, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-4102", coverURL: "cover-v-4102"},
		{contentID: 4101, authorID: 2201, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-4101", coverURL: "cover-a-4101"},
	})

	listCalls := 0
	logic := NewFollowFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		FollowRpc: &fakeFollowService{
			listFolloweesFunc: func(ctx context.Context, in *followservice.ListFolloweesReq, opts ...grpc.CallOption) (*followservice.ListFolloweesRes, error) {
				listCalls++
				return &followservice.ListFolloweesRes{
					FollowUserIds: []int64{2201, 2202},
					HasMore:       false,
					NextCursor:    0,
				}, nil
			},
		},
	})

	resp, err := logic.FollowFeed(&contentpb.FollowFeedReq{
		UserId:   1201,
		Cursor:   "",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("FollowFeed rebuild returned error: %v", err)
	}
	if listCalls != 1 {
		t.Fatalf("listCalls after rebuild = %d, want 1", listCalls)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) after rebuild = %d, want 3", len(resp.GetItems()))
	}
	if resp.GetItems()[0].GetContentId() != 4103 || resp.GetItems()[1].GetContentId() != 4102 || resp.GetItems()[2].GetContentId() != 4101 {
		t.Fatalf("rebuild order = [%d %d %d], want [4103 4102 4101]",
			resp.GetItems()[0].GetContentId(), resp.GetItems()[1].GetContentId(), resp.GetItems()[2].GetContentId())
	}

	inboxKey := redisconsts.BuildFollowInboxKey(1201)
	if !store.Exists(inboxKey) {
		t.Fatalf("expected rebuilt inbox key %s to exist", inboxKey)
	}

	resp, err = logic.FollowFeed(&contentpb.FollowFeedReq{
		UserId:   1201,
		Cursor:   "",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("FollowFeed cache hit returned error: %v", err)
	}
	if listCalls != 1 {
		t.Fatalf("listCalls after cache hit = %d, want still 1", listCalls)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) on cache hit = %d, want 3", len(resp.GetItems()))
	}
}
