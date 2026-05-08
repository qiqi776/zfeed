package likelogic

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"
)

type likeTestRow struct {
	ID            int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID        int64 `gorm:"column:user_id"`
	Scene         int32 `gorm:"column:scene"`
	ContentID     int64 `gorm:"column:content_id"`
	ContentUserID int64 `gorm:"column:content_user_id"`
	Status        int32 `gorm:"column:status"`
	LastEventTs   int64 `gorm:"column:last_event_ts"`
	IsDeleted     int32 `gorm:"column:is_deleted"`
}

func (likeTestRow) TableName() string {
	return "zfeed_like"
}

func newLikeLogicTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&likeTestRow{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func newLikeLogicTestRedis(t *testing.T) *gzredis.Redis {
	t.Helper()

	store := miniredis.RunT(t)
	return gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
}

func TestQueryLikeInfoReturnsCountAndState(t *testing.T) {
	db := newLikeLogicTestDB(t)
	redisClient := newLikeLogicTestRedis(t)

	rows := []likeTestRow{
		{UserID: 1001, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9001, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1002, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9001, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1003, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9001, ContentUserID: 2001, Status: 20, IsDeleted: 0},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	logic := NewQueryLikeInfoLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.QueryLikeInfo(&interaction.QueryLikeInfoReq{
		UserId:    1001,
		ContentId: 9001,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("QueryLikeInfo returned error: %v", err)
	}
	if resp.GetLikeCount() != 2 {
		t.Fatalf("like_count = %d, want 2", resp.GetLikeCount())
	}
	if !resp.GetIsLiked() {
		t.Fatal("is_liked = false, want true")
	}

	cacheValues, err := redisClient.HmgetCtx(context.Background(), likeCacheKey(1001), likeTargetKey(interaction.Scene_ARTICLE, 9001))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if len(cacheValues) != 1 || cacheValues[0] != likeCacheValueLiked {
		t.Fatalf("cache value = %v, want [%s]", cacheValues, likeCacheValueLiked)
	}

	ttl, err := redisClient.TtlCtx(context.Background(), likeCacheKey(1001))
	if err != nil {
		t.Fatalf("read cache ttl: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("cache ttl = %d, want > 0", ttl)
	}
}

func TestBatchQueryLikeInfoMergesCacheAndDB(t *testing.T) {
	db := newLikeLogicTestDB(t)
	redisClient := newLikeLogicTestRedis(t)

	rows := []likeTestRow{
		{UserID: 1001, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9101, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1002, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9101, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1001, Scene: int32(interaction.Scene_VIDEO), ContentID: 9102, ContentUserID: 2002, Status: 10, IsDeleted: 0},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}
	if err := redisClient.HsetCtx(context.Background(), "like:user:1001", likeTargetKey(interaction.Scene_ARTICLE, 9101), "1"); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	logic := NewBatchLikeInfoLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.BatchLikeInfo(&interaction.BatchLikeInfoReq{
		UserId: 1001,
		LikeInfos: []*interaction.LikeInfo{
			{ContentId: 9101, Scene: interaction.Scene_ARTICLE},
			{ContentId: 9102, Scene: interaction.Scene_VIDEO},
			{ContentId: 9103, Scene: interaction.Scene_ARTICLE},
		},
	})
	if err != nil {
		t.Fatalf("BatchQueryLikeInfo returned error: %v", err)
	}
	if len(resp.GetLikeInfos()) != 3 {
		t.Fatalf("len(like_infos) = %d, want 3", len(resp.GetLikeInfos()))
	}

	assertLikeInfo(t, resp.GetLikeInfos()[0], 9101, 2, true)
	assertLikeInfo(t, resp.GetLikeInfos()[1], 9102, 1, true)
	assertLikeInfo(t, resp.GetLikeInfos()[2], 9103, 0, false)
}

func TestQueryLikeInfoPrefersExplicitUnlikeCache(t *testing.T) {
	db := newLikeLogicTestDB(t)
	redisClient := newLikeLogicTestRedis(t)

	rows := []likeTestRow{
		{UserID: 1001, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9201, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1002, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9201, ContentUserID: 2001, Status: 10, IsDeleted: 0},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}
	if err := cacheLikeState(context.Background(), redisClient, likeCacheKey(1001), likeTargetKey(interaction.Scene_ARTICLE, 9201), false); err != nil {
		t.Fatalf("seed unlike cache: %v", err)
	}

	logic := NewQueryLikeInfoLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.QueryLikeInfo(&interaction.QueryLikeInfoReq{
		UserId:    1001,
		ContentId: 9201,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("QueryLikeInfo returned error: %v", err)
	}
	if resp.GetLikeCount() != 2 {
		t.Fatalf("like_count = %d, want 2", resp.GetLikeCount())
	}
	if resp.GetIsLiked() {
		t.Fatal("is_liked = true, want false when explicit unlike cache exists")
	}
}

func TestBatchLikeInfoPrefersExplicitUnlikeCache(t *testing.T) {
	db := newLikeLogicTestDB(t)
	redisClient := newLikeLogicTestRedis(t)

	rows := []likeTestRow{
		{UserID: 1001, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9301, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1002, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9301, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1001, Scene: int32(interaction.Scene_VIDEO), ContentID: 9302, ContentUserID: 2002, Status: 10, IsDeleted: 0},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}
	if err := cacheLikeState(context.Background(), redisClient, likeCacheKey(1001), likeTargetKey(interaction.Scene_ARTICLE, 9301), false); err != nil {
		t.Fatalf("seed unlike cache: %v", err)
	}
	if err := cacheLikeState(context.Background(), redisClient, likeCacheKey(1001), likeTargetKey(interaction.Scene_VIDEO, 9302), true); err != nil {
		t.Fatalf("seed like cache: %v", err)
	}

	logic := NewBatchLikeInfoLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.BatchLikeInfo(&interaction.BatchLikeInfoReq{
		UserId: 1001,
		LikeInfos: []*interaction.LikeInfo{
			{ContentId: 9301, Scene: interaction.Scene_ARTICLE},
			{ContentId: 9302, Scene: interaction.Scene_VIDEO},
		},
	})
	if err != nil {
		t.Fatalf("BatchLikeInfo returned error: %v", err)
	}
	if len(resp.GetLikeInfos()) != 2 {
		t.Fatalf("len(like_infos) = %d, want 2", len(resp.GetLikeInfos()))
	}

	assertLikeInfo(t, resp.GetLikeInfos()[0], 9301, 2, false)
	assertLikeInfo(t, resp.GetLikeInfos()[1], 9302, 1, true)
}

func TestLikeQueriesSeparateSameContentIDByScene(t *testing.T) {
	db := newLikeLogicTestDB(t)
	redisClient := newLikeLogicTestRedis(t)

	rows := []likeTestRow{
		{UserID: 1001, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9401, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1002, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9401, ContentUserID: 2001, Status: 10, IsDeleted: 0},
		{UserID: 1001, Scene: int32(interaction.Scene_COMMENT), ContentID: 9401, ContentUserID: 3001, Status: 10, IsDeleted: 0},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	logic := NewBatchLikeInfoLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.BatchLikeInfo(&interaction.BatchLikeInfoReq{
		UserId: 1001,
		LikeInfos: []*interaction.LikeInfo{
			{ContentId: 9401, Scene: interaction.Scene_ARTICLE},
			{ContentId: 9401, Scene: interaction.Scene_COMMENT},
		},
	})
	if err != nil {
		t.Fatalf("BatchLikeInfo returned error: %v", err)
	}
	if len(resp.GetLikeInfos()) != 2 {
		t.Fatalf("len(like_infos) = %d, want 2", len(resp.GetLikeInfos()))
	}

	assertLikeInfo(t, resp.GetLikeInfos()[0], 9401, 2, true)
	if resp.GetLikeInfos()[0].GetScene() != interaction.Scene_ARTICLE {
		t.Fatalf("first scene = %s, want ARTICLE", resp.GetLikeInfos()[0].GetScene().String())
	}
	assertLikeInfo(t, resp.GetLikeInfos()[1], 9401, 1, true)
	if resp.GetLikeInfos()[1].GetScene() != interaction.Scene_COMMENT {
		t.Fatalf("second scene = %s, want COMMENT", resp.GetLikeInfos()[1].GetScene().String())
	}
}

func assertLikeInfo(t *testing.T, item *interaction.QueryLikeInfoRes, contentID int64, count int64, isLiked bool) {
	t.Helper()

	if item == nil {
		t.Fatal("like info item is nil")
	}
	if item.GetContentId() != contentID {
		t.Fatalf("content_id = %d, want %d", item.GetContentId(), contentID)
	}
	if item.GetLikeCount() != count {
		t.Fatalf("like_count = %d, want %d", item.GetLikeCount(), count)
	}
	if item.GetIsLiked() != isLiked {
		t.Fatalf("is_liked = %v, want %v", item.GetIsLiked(), isLiked)
	}
}
