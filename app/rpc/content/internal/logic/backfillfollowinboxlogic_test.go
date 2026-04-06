package logic

import (
	"context"
	"strconv"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/svc"
)

func TestBackfillFollowInbox_UsesPublishZSet(t *testing.T) {
	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	db, err := gorm.Open(sqlite.Open("file:backfill_follow_inbox?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedContent{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	publishKey := redisconsts.BuildUserPublishKey(2002)
	for _, contentID := range []int64{3003, 3002, 3001} {
		member := strconv.FormatInt(contentID, 10)
		store.ZAdd(publishKey, float64(contentID), member)
	}

	logic := NewBackfillFollowInboxLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	resp, err := logic.BackfillFollowInbox(&contentpb.BackfillFollowInboxReq{
		FollowerId: 1001,
		FolloweeId: 2002,
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("BackfillFollowInbox returned error: %v", err)
	}
	if resp.GetAddedCount() != 2 {
		t.Fatalf("added_count = %d, want 2", resp.GetAddedCount())
	}

	inboxKey := redisconsts.BuildFollowInboxKey(1001)
	members, err := store.ZMembers(inboxKey)
	if err != nil {
		t.Fatalf("redis zmembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("len(members) = %d, want 2", len(members))
	}
	if members[0] != "3002" || members[1] != "3003" {
		t.Fatalf("members = %v, want [3002 3003] in ascending zset view", members)
	}
}
