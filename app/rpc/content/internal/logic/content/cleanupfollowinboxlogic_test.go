package contentlogic

import (
	"context"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
	followservice "zfeed/app/rpc/interaction/client/followservice"
)

func TestCleanupRemove(t *testing.T) {
	store, redisClient := newCleanupRedis(t)
	db := newContentServiceTestDB(t)

	seedCleanupContent(t, db, []cleanupSeed{
		{contentID: 6103, authorID: 2001},
		{contentID: 6102, authorID: 2002},
		{contentID: 6101, authorID: 2001},
	})

	inboxKey := redisconsts.BuildFollowInboxKey(1001)
	for _, contentID := range []int64{6103, 6102, 6101} {
		member := strconv.FormatInt(contentID, 10)
		store.ZAdd(inboxKey, float64(contentID), member)
	}

	logic := NewCleanupFollowInboxLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		FollowRpc: &stubFollowService{
			batchQueryFollowing: func(ctx context.Context, in *followservice.BatchQueryFollowingReq, opts ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error) {
				return &followservice.BatchQueryFollowingRes{
					Items: []*followservice.FollowingState{
						{UserId: 2001, IsFollowing: false},
					},
				}, nil
			},
		},
	})

	resp, err := logic.CleanupFollowInbox(&contentpb.CleanupFollowInboxReq{
		FollowerId: 1001,
		FolloweeId: 2001,
	})
	if err != nil {
		t.Fatalf("CleanupFollowInbox returned error: %v", err)
	}
	if resp.GetRemovedCount() != 2 || resp.GetSkipped() {
		t.Fatalf("cleanup resp = %+v, want removed=2 skipped=false", resp)
	}

	members, err := store.ZMembers(inboxKey)
	if err != nil {
		t.Fatalf("redis zmembers: %v", err)
	}
	if len(members) != 1 || members[0] != "6102" {
		t.Fatalf("members after cleanup = %v, want [6102]", members)
	}
}

func TestCleanupSkip(t *testing.T) {
	store, redisClient := newCleanupRedis(t)
	db := newContentServiceTestDB(t)

	seedCleanupContent(t, db, []cleanupSeed{
		{contentID: 6201, authorID: 2001},
	})

	inboxKey := redisconsts.BuildFollowInboxKey(1001)
	store.ZAdd(inboxKey, 6201, "6201")

	logic := NewCleanupFollowInboxLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		FollowRpc: &stubFollowService{
			batchQueryFollowing: func(ctx context.Context, in *followservice.BatchQueryFollowingReq, opts ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error) {
				return &followservice.BatchQueryFollowingRes{
					Items: []*followservice.FollowingState{
						{UserId: 2001, IsFollowing: true},
					},
				}, nil
			},
		},
	})

	resp, err := logic.CleanupFollowInbox(&contentpb.CleanupFollowInboxReq{
		FollowerId: 1001,
		FolloweeId: 2001,
	})
	if err != nil {
		t.Fatalf("CleanupFollowInbox returned error: %v", err)
	}
	if resp.GetRemovedCount() != 0 || !resp.GetSkipped() {
		t.Fatalf("cleanup resp = %+v, want removed=0 skipped=true", resp)
	}

	members, err := store.ZMembers(inboxKey)
	if err != nil {
		t.Fatalf("redis zmembers: %v", err)
	}
	if len(members) != 1 || members[0] != "6201" {
		t.Fatalf("members after skip = %v, want [6201]", members)
	}
}

type cleanupSeed struct {
	contentID int64
	authorID  int64
}

func newCleanupRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	return store, client
}

func seedCleanupContent(t *testing.T, db *gorm.DB, rows []cleanupSeed) {
	t.Helper()

	for _, row := range rows {
		publishedAt := time.Unix(row.contentID, 0)
		if err := db.Create(&contentServiceTestContent{
			ID:          row.contentID,
			UserID:      row.authorID,
			ContentType: int32(contentpb.ContentType_ARTICLE),
			Status:      int32(contentpb.ContentStatus_PUBLISHED),
			Visibility:  int32(contentpb.Visibility_PUBLIC),
			PublishedAt: &publishedAt,
			IsDeleted:   0,
		}).Error; err != nil {
			t.Fatalf("create content row %d: %v", row.contentID, err)
		}
	}
}
