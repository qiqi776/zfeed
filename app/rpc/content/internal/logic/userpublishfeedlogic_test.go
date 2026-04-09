package logic

import (
	"context"
	"strconv"
	"testing"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
)

func TestUserPublishFeed_HitReturnsOrderedItems(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 5103, authorID: 2001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-5103", coverURL: "cover-a-5103"},
		{contentID: 5102, authorID: 2001, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-5102", coverURL: "cover-v-5102"},
		{contentID: 5101, authorID: 2001, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-5101", coverURL: "cover-a-5101"},
	})

	feedKey := redisconsts.BuildUserPublishFeedKey(2001)
	for _, contentID := range []int64{5103, 5102, 5101} {
		store.ZAdd(feedKey, float64(contentID), strconv.FormatInt(contentID, 10))
	}

	logic := NewUserPublishFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.UserPublishFeed(&contentpb.UserPublishFeedReq{
		AuthorId: 2001,
		Cursor:   "",
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("UserPublishFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(resp.GetItems()))
	}
	if resp.GetItems()[0].GetContentId() != 5103 || resp.GetItems()[1].GetContentId() != 5102 || resp.GetItems()[2].GetContentId() != 5101 {
		t.Fatalf("content ids = [%d %d %d], want [5103 5102 5101]",
			resp.GetItems()[0].GetContentId(), resp.GetItems()[1].GetContentId(), resp.GetItems()[2].GetContentId())
	}
	if resp.GetItems()[0].GetLikeCount() != 0 {
		t.Fatalf("like_count = %d, want 0", resp.GetItems()[0].GetLikeCount())
	}
}

func TestUserPublishFeed_CursorPagination(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 5203, authorID: 2002, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-5203", coverURL: "cover-a-5203"},
		{contentID: 5202, authorID: 2002, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-5202", coverURL: "cover-v-5202"},
		{contentID: 5201, authorID: 2002, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-5201", coverURL: "cover-a-5201"},
	})

	feedKey := redisconsts.BuildUserPublishFeedKey(2002)
	for _, contentID := range []int64{5203, 5202, 5201} {
		store.ZAdd(feedKey, float64(contentID), strconv.FormatInt(contentID, 10))
	}

	logic := NewUserPublishFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	firstPage, err := logic.UserPublishFeed(&contentpb.UserPublishFeedReq{
		AuthorId: 2002,
		Cursor:   "",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("first page returned error: %v", err)
	}
	if !firstPage.GetHasMore() || firstPage.GetNextCursor() != "5202" {
		t.Fatalf("first page pagination = has_more:%v next_cursor:%s, want true/5202", firstPage.GetHasMore(), firstPage.GetNextCursor())
	}

	secondPage, err := logic.UserPublishFeed(&contentpb.UserPublishFeedReq{
		AuthorId: 2002,
		Cursor:   firstPage.GetNextCursor(),
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("second page returned error: %v", err)
	}
	if len(secondPage.GetItems()) != 1 || secondPage.GetItems()[0].GetContentId() != 5201 {
		t.Fatalf("second page items = %+v, want [5201]", secondPage.GetItems())
	}
}

func TestUserPublishFeed_MissRebuildsFromDB(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 5303, authorID: 2003, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-5303", coverURL: "cover-a-5303"},
		{contentID: 5302, authorID: 2003, contentType: contentpb.ContentType_CONTENT_TYPE_VIDEO, title: "video-5302", coverURL: "cover-v-5302"},
		{contentID: 5301, authorID: 2003, contentType: contentpb.ContentType_CONTENT_TYPE_ARTICLE, title: "article-5301", coverURL: "cover-a-5301"},
	})

	logic := NewUserPublishFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.UserPublishFeed(&contentpb.UserPublishFeedReq{
		AuthorId: 2003,
		Cursor:   "",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("UserPublishFeed rebuild returned error: %v", err)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(resp.GetItems()))
	}

	feedKey := redisconsts.BuildUserPublishFeedKey(2003)
	if !store.Exists(feedKey) {
		t.Fatalf("expected publish feed key %s to exist after rebuild", feedKey)
	}
}
