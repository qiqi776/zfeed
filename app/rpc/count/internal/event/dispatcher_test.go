package event

import (
	"context"
	"strconv"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/changeevent"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
	"zfeed/app/rpc/count/internal/logic"
	"zfeed/app/rpc/count/internal/model"
	"zfeed/app/rpc/count/internal/svc"
)

type dispatcherTestContent struct {
	ID         int64 `gorm:"column:id;primaryKey"`
	UserID     int64 `gorm:"column:user_id"`
	Status     int32 `gorm:"column:status"`
	Visibility int32 `gorm:"column:visibility"`
	IsDeleted  int32 `gorm:"column:is_deleted"`
}

func (dispatcherTestContent) TableName() string {
	return "zfeed_content"
}

func TestDispatcherLikeEventUpdatesCountAndInvalidatesCache(t *testing.T) {
	svcCtx := newDispatcherTestServiceContext(t)
	ctx := context.Background()

	countCacheKey := redisconsts.BuildCountValueKey(
		strconv.FormatInt(int64(count.BizType_LIKE), 10),
		strconv.FormatInt(int64(count.TargetType_CONTENT), 10),
		"9001",
	)
	userProfileCacheKey := redisconsts.BuildUserProfileCountsKey(3001)
	if err := svcCtx.Redis.SetexCtx(ctx, countCacheKey, "99", 300); err != nil {
		t.Fatalf("seed count cache: %v", err)
	}
	if err := svcCtx.Redis.SetexCtx(ctx, userProfileCacheKey, `{"like_count":99}`, 300); err != nil {
		t.Fatalf("seed user profile cache: %v", err)
	}

	dispatcher := NewDispatcher(ctx, svcCtx, "count.dispatcher.test")
	evt := changeevent.ChangeEvent{
		EventID:   "like-insert-9001",
		Source:    "mock",
		Table:     "zfeed_like",
		Operation: "INSERT",
		Current: map[string]any{
			"id":              1,
			"content_id":      9001,
			"content_user_id": 3001,
			"status":          10,
		},
	}

	applied, err := dispatcher.Dispatch(ctx, evt)
	if err != nil {
		t.Fatalf("dispatch like event: %v", err)
	}
	if applied != 1 {
		t.Fatalf("applied updates = %d, want 1", applied)
	}
	incKey := redisconsts.BuildHotFeedIncKey(int(9001 % int64(redisconsts.RedisFeedHotIncDefaultShards)))
	incMap, err := svcCtx.Redis.HgetallCtx(ctx, incKey)
	if err != nil {
		t.Fatalf("read hot increment bucket: %v", err)
	}
	if incMap["9001"] != "1" {
		t.Fatalf("hot increment bucket value = %q, want %q", incMap["9001"], "1")
	}

	applied, err = dispatcher.Dispatch(ctx, evt)
	if err != nil {
		t.Fatalf("dispatch duplicate like event: %v", err)
	}
	if applied != 0 {
		t.Fatalf("duplicate applied updates = %d, want 0", applied)
	}

	getCountLogic := logic.NewGetCountLogic(ctx, svcCtx)
	getCountResp, err := getCountLogic.GetCount(&count.GetCountReq{
		BizType:    count.BizType_LIKE,
		TargetType: count.TargetType_CONTENT,
		TargetId:   9001,
	})
	if err != nil {
		t.Fatalf("get count after dispatch: %v", err)
	}
	if getCountResp.GetValue() != 1 {
		t.Fatalf("content like count = %d, want 1", getCountResp.GetValue())
	}

	getProfileLogic := logic.NewGetUserProfileCountsLogic(ctx, svcCtx)
	getProfileResp, err := getProfileLogic.GetUserProfileCounts(&count.GetUserProfileCountsReq{UserId: 3001})
	if err != nil {
		t.Fatalf("get user profile counts after dispatch: %v", err)
	}
	if getProfileResp.GetLikeCount() != 1 {
		t.Fatalf("user profile like count = %d, want 1", getProfileResp.GetLikeCount())
	}

	if cached, err := svcCtx.Redis.GetCtx(ctx, countCacheKey); err != nil {
		t.Fatalf("read rebuilt count cache: %v", err)
	} else if cached != "1" {
		t.Fatalf("rebuilt count cache = %q, want %q", cached, "1")
	}

	if cached, err := svcCtx.Redis.GetCtx(ctx, userProfileCacheKey); err != nil {
		t.Fatalf("read rebuilt user profile cache: %v", err)
	} else if cached == "" {
		t.Fatal("user profile cache should be rebuilt after read")
	}
}

func TestDispatcherFollowEventUpdatesBothUserCounts(t *testing.T) {
	svcCtx := newDispatcherTestServiceContext(t)
	ctx := context.Background()

	dispatcher := NewDispatcher(ctx, svcCtx, "count.dispatcher.test")
	evt := changeevent.ChangeEvent{
		EventID:   "follow-insert-7001-8001",
		Source:    "mock",
		Table:     "zfeed_follow",
		Operation: "INSERT",
		Current: map[string]any{
			"id":             1,
			"user_id":        7001,
			"follow_user_id": 8001,
			"status":         10,
			"is_deleted":     0,
		},
	}

	applied, err := dispatcher.Dispatch(ctx, evt)
	if err != nil {
		t.Fatalf("dispatch follow event: %v", err)
	}
	if applied != 2 {
		t.Fatalf("applied updates = %d, want 2", applied)
	}

	getCountLogic := logic.NewGetCountLogic(ctx, svcCtx)
	followingResp, err := getCountLogic.GetCount(&count.GetCountReq{
		BizType:    count.BizType_FOLLOWING,
		TargetType: count.TargetType_USER,
		TargetId:   7001,
	})
	if err != nil {
		t.Fatalf("get following count: %v", err)
	}
	if followingResp.GetValue() != 1 {
		t.Fatalf("following count = %d, want 1", followingResp.GetValue())
	}

	followedResp, err := getCountLogic.GetCount(&count.GetCountReq{
		BizType:    count.BizType_FOLLOWED,
		TargetType: count.TargetType_USER,
		TargetId:   8001,
	})
	if err != nil {
		t.Fatalf("get followed count: %v", err)
	}
	if followedResp.GetValue() != 1 {
		t.Fatalf("followed count = %d, want 1", followedResp.GetValue())
	}
}

func newDispatcherTestServiceContext(t *testing.T) *svc.ServiceContext {
	t.Helper()

	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	db, err := gorm.Open(sqlite.Open("file:count_dispatcher_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedCountValue{}, &model.ZfeedMqConsumeDedup{}, &dispatcherTestContent{}); err != nil {
		t.Fatalf("auto migrate count models: %v", err)
	}

	return &svc.ServiceContext{
		Redis:   redisClient,
		MysqlDb: db,
	}
}
