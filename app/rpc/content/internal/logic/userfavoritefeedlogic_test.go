package logic

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
	favoriteservice "zfeed/app/rpc/interaction/client/favoriteservice"
)

var _ favoriteservice.FavoriteService = (*fakeFavoriteService)(nil)

type fakeFavoriteService struct {
	queryFavoriteListFunc func(ctx context.Context, in *favoriteservice.QueryFavoriteListReq, opts ...grpc.CallOption) (*favoriteservice.QueryFavoriteListRes, error)
}

func (f *fakeFavoriteService) Favorite(context.Context, *favoriteservice.FavoriteReq, ...grpc.CallOption) (*favoriteservice.FavoriteRes, error) {
	return nil, errors.New("unexpected Favorite call")
}

func (f *fakeFavoriteService) RemoveFavorite(context.Context, *favoriteservice.RemoveFavoriteReq, ...grpc.CallOption) (*favoriteservice.RemoveFavoriteRes, error) {
	return nil, errors.New("unexpected RemoveFavorite call")
}

func (f *fakeFavoriteService) QueryFavoriteInfo(context.Context, *favoriteservice.QueryFavoriteInfoReq, ...grpc.CallOption) (*favoriteservice.QueryFavoriteInfoRes, error) {
	return nil, errors.New("unexpected QueryFavoriteInfo call")
}

func (f *fakeFavoriteService) QueryFavoriteList(ctx context.Context, in *favoriteservice.QueryFavoriteListReq, opts ...grpc.CallOption) (*favoriteservice.QueryFavoriteListRes, error) {
	if f.queryFavoriteListFunc == nil {
		return nil, errors.New("unexpected QueryFavoriteList call")
	}
	return f.queryFavoriteListFunc(ctx, in, opts...)
}

func TestUserFavoriteFeed_HitReturnsOrderedItems(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 6103, authorID: 3001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-6103", coverURL: "cover-a-6103"},
		{contentID: 6102, authorID: 3002, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-6102", coverURL: "cover-v-6102"},
		{contentID: 6101, authorID: 3003, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-6101", coverURL: "cover-a-6101"},
	})

	feedKey := redisconsts.BuildUserFavoriteFeedKey(1001)
	store.ZAdd(feedKey, 9003, "6103")
	store.ZAdd(feedKey, 9002, "6102")
	store.ZAdd(feedKey, 9001, "6101")

	logic := NewUserFavoriteFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.UserFavoriteFeed(&contentpb.UserFavoriteFeedReq{
		UserId:   1001,
		Cursor:   "",
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("UserFavoriteFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(resp.GetItems()))
	}
	if resp.GetItems()[0].GetContentId() != 6103 || resp.GetItems()[1].GetContentId() != 6102 || resp.GetItems()[2].GetContentId() != 6101 {
		t.Fatalf("content ids = [%d %d %d], want [6103 6102 6101]",
			resp.GetItems()[0].GetContentId(), resp.GetItems()[1].GetContentId(), resp.GetItems()[2].GetContentId())
	}
}

func TestUserFavoriteFeed_CursorPagination(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 6203, authorID: 3001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-6203", coverURL: "cover-a-6203"},
		{contentID: 6202, authorID: 3002, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-6202", coverURL: "cover-v-6202"},
		{contentID: 6201, authorID: 3003, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-6201", coverURL: "cover-a-6201"},
	})

	feedKey := redisconsts.BuildUserFavoriteFeedKey(1002)
	store.ZAdd(feedKey, 9103, "6203")
	store.ZAdd(feedKey, 9102, "6202")
	store.ZAdd(feedKey, 9101, "6201")

	logic := NewUserFavoriteFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	firstPage, err := logic.UserFavoriteFeed(&contentpb.UserFavoriteFeedReq{
		UserId:   1002,
		Cursor:   "",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("first page returned error: %v", err)
	}
	if !firstPage.GetHasMore() || firstPage.GetNextCursor() != "9102" {
		t.Fatalf("first page pagination = has_more:%v next_cursor:%s, want true/9102", firstPage.GetHasMore(), firstPage.GetNextCursor())
	}

	secondPage, err := logic.UserFavoriteFeed(&contentpb.UserFavoriteFeedReq{
		UserId:   1002,
		Cursor:   firstPage.GetNextCursor(),
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("second page returned error: %v", err)
	}
	if len(secondPage.GetItems()) != 1 || secondPage.GetItems()[0].GetContentId() != 6201 {
		t.Fatalf("second page items = %+v, want [6201]", secondPage.GetItems())
	}
}

func TestUserFavoriteFeed_MissRebuildsFromFavoriteRPC(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 6303, authorID: 3001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-6303", coverURL: "cover-a-6303"},
		{contentID: 6302, authorID: 3002, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-6302", coverURL: "cover-v-6302"},
		{contentID: 6301, authorID: 3003, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-6301", coverURL: "cover-a-6301"},
	})

	queryCalls := 0
	logic := NewUserFavoriteFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		FavoriteRpc: &fakeFavoriteService{
			queryFavoriteListFunc: func(ctx context.Context, in *favoriteservice.QueryFavoriteListReq, opts ...grpc.CallOption) (*favoriteservice.QueryFavoriteListRes, error) {
				queryCalls++
				return &favoriteservice.QueryFavoriteListRes{
					Items: []*favoriteservice.FavoriteItem{
						{FavoriteId: 9203, ContentId: 6303, ContentUserId: 3001},
						{FavoriteId: 9202, ContentId: 6302, ContentUserId: 3002},
						{FavoriteId: 9201, ContentId: 6301, ContentUserId: 3003},
					},
					HasMore:    false,
					NextCursor: 0,
				}, nil
			},
		},
	})

	resp, err := logic.UserFavoriteFeed(&contentpb.UserFavoriteFeedReq{
		UserId:   1003,
		Cursor:   "",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("UserFavoriteFeed rebuild returned error: %v", err)
	}
	if queryCalls != 1 {
		t.Fatalf("queryCalls = %d, want 1", queryCalls)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(resp.GetItems()))
	}

	feedKey := redisconsts.BuildUserFavoriteFeedKey(1003)
	if !store.Exists(feedKey) {
		t.Fatalf("expected favorite feed key %s to exist after rebuild", feedKey)
	}
	score, err := store.ZScore(feedKey, "6302")
	if err != nil {
		t.Fatalf("zscore favorite feed: %v", err)
	}
	if score != 9202 {
		t.Fatalf("score for content 6302 = %v, want 9202", score)
	}
}

func TestUserFavoriteFeed_SkipsDirtyContentIDs(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 6402, authorID: 3002, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-6402", coverURL: "cover-v-6402"},
	})

	feedKey := redisconsts.BuildUserFavoriteFeedKey(1004)
	store.ZAdd(feedKey, 9302, "6402")
	store.ZAdd(feedKey, 9301, "6499")

	logic := NewUserFavoriteFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.UserFavoriteFeed(&contentpb.UserFavoriteFeedReq{
		UserId:   1004,
		Cursor:   "",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("UserFavoriteFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetContentId() != 6402 {
		t.Fatalf("items = %+v, want only content 6402", resp.GetItems())
	}
}
