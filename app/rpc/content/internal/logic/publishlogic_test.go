package logic

import (
	"context"
	"strconv"
	"strings"
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

func newTestDB(t *testing.T, migrateModels ...any) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if len(migrateModels) > 0 {
		if err := db.AutoMigrate(migrateModels...); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
	}

	return db
}

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	return store, client
}

func TestPublishArticle_PersistsRowsAndUpdatesCache(t *testing.T) {
	db := newTestDB(t, &model.ZfeedContent{}, &model.ZfeedArticle{}, &model.ZfeedVideo{})
	store, client := newTestRedis(t)
	defer store.Close()

	logic := NewPublishArticleLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	resp, err := logic.PublishArticle(&contentpb.ArticlePublishReq{
		UserId:     101,
		Title:      "article-title",
		Cover:      "https://example.com/a.png",
		Content:    "article-body",
		Visibility: contentpb.Visibility_VISIBILITY_PUBLIC,
	})
	if err != nil {
		t.Fatalf("PublishArticle returned error: %v", err)
	}
	if resp.GetContentId() <= 0 {
		t.Fatalf("content_id = %d, want > 0", resp.GetContentId())
	}

	var contentRow model.ZfeedContent
	if err := db.First(&contentRow, resp.GetContentId()).Error; err != nil {
		t.Fatalf("query zfeed_content: %v", err)
	}
	if contentRow.ContentType != int32(contentpb.ContentType_CONTENT_TYPE_ARTICLE) {
		t.Fatalf("content_type = %d, want %d", contentRow.ContentType, int32(contentpb.ContentType_CONTENT_TYPE_ARTICLE))
	}

	var articleRow model.ZfeedArticle
	if err := db.Where("content_id = ?", resp.GetContentId()).First(&articleRow).Error; err != nil {
		t.Fatalf("query zfeed_article: %v", err)
	}
	if articleRow.Title != "article-title" {
		t.Fatalf("article title = %q, want %q", articleRow.Title, "article-title")
	}

	key := redisconsts.BuildUserPublishKey(101)
	if !store.Exists(key) {
		t.Fatalf("redis key %q does not exist", key)
	}

	member := strconv.FormatInt(resp.GetContentId(), 10)
	members, err := store.ZMembers(key)
	if err != nil {
		t.Fatalf("redis zmembers: %v", err)
	}
	if len(members) != 1 || members[0] != member {
		t.Fatalf("zmembers = %v, want [%s]", members, member)
	}

	score, err := store.ZScore(key, member)
	if err != nil {
		t.Fatalf("redis zscore: %v", err)
	}
	if score != float64(resp.GetContentId()) {
		t.Fatalf("zscore = %v, want %d", score, resp.GetContentId())
	}
}

func TestPublishVideo_PersistsRowsAndUpdatesCache(t *testing.T) {
	db := newTestDB(t, &model.ZfeedContent{}, &model.ZfeedArticle{}, &model.ZfeedVideo{})
	store, client := newTestRedis(t)
	defer store.Close()

	logic := NewPublishVideoLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	resp, err := logic.PublishVideo(&contentpb.VideoPublishReq{
		UserId:      202,
		Title:       "video-title",
		OriginUrl:   "https://example.com/v.mp4",
		CoverUrl:    "https://example.com/v.png",
		Duration:    120,
		Visibility:  contentpb.Visibility_VISIBILITY_PUBLIC,
		Description: nil,
	})
	if err != nil {
		t.Fatalf("PublishVideo returned error: %v", err)
	}
	if resp.GetContentId() <= 0 {
		t.Fatalf("content_id = %d, want > 0", resp.GetContentId())
	}

	var contentRow model.ZfeedContent
	if err := db.First(&contentRow, resp.GetContentId()).Error; err != nil {
		t.Fatalf("query zfeed_content: %v", err)
	}
	if contentRow.ContentType != int32(contentpb.ContentType_CONTENT_TYPE_VIDEO) {
		t.Fatalf("content_type = %d, want %d", contentRow.ContentType, int32(contentpb.ContentType_CONTENT_TYPE_VIDEO))
	}

	var videoRow model.ZfeedVideo
	if err := db.Where("content_id = ?", resp.GetContentId()).First(&videoRow).Error; err != nil {
		t.Fatalf("query zfeed_video: %v", err)
	}
	if videoRow.Title != "video-title" {
		t.Fatalf("video title = %q, want %q", videoRow.Title, "video-title")
	}

	key := redisconsts.BuildUserPublishKey(202)
	member := strconv.FormatInt(resp.GetContentId(), 10)

	score, err := store.ZScore(key, member)
	if err != nil {
		t.Fatalf("redis zscore: %v", err)
	}
	if score != float64(resp.GetContentId()) {
		t.Fatalf("zscore = %v, want %d", score, resp.GetContentId())
	}
}

func TestPublishArticle_RollsBackWhenSubTableInsertFails(t *testing.T) {
	db := newTestDB(t, &model.ZfeedContent{})
	store, client := newTestRedis(t)
	defer store.Close()

	logic := NewPublishArticleLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	_, err := logic.PublishArticle(&contentpb.ArticlePublishReq{
		UserId:     303,
		Title:      "rollback-article",
		Cover:      "https://example.com/rollback.png",
		Content:    "body",
		Visibility: contentpb.Visibility_VISIBILITY_PUBLIC,
	})
	if err == nil {
		t.Fatal("PublishArticle error = nil, want not nil")
	}

	var count int64
	if err := db.Model(&model.ZfeedContent{}).Count(&count).Error; err != nil {
		t.Fatalf("count zfeed_content: %v", err)
	}
	if count != 0 {
		t.Fatalf("zfeed_content row count = %d, want 0", count)
	}

	key := redisconsts.BuildUserPublishKey(303)
	if store.Exists(key) {
		t.Fatalf("redis key %q should not exist after rollback", key)
	}
}

func TestPublishArticle_DoesNotFailWhenRedisUpdateFails(t *testing.T) {
	db := newTestDB(t, &model.ZfeedContent{}, &model.ZfeedArticle{})

	store, client := newTestRedis(t)
	store.Close() // simulate redis failure after client has been created

	logic := NewPublishArticleLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	resp, err := logic.PublishArticle(&contentpb.ArticlePublishReq{
		UserId:     404,
		Title:      "best-effort-cache",
		Cover:      "https://example.com/cache.png",
		Content:    "body",
		Visibility: contentpb.Visibility_VISIBILITY_PUBLIC,
	})
	if err != nil {
		t.Fatalf("PublishArticle returned error: %v", err)
	}
	if resp.GetContentId() <= 0 {
		t.Fatalf("content_id = %d, want > 0", resp.GetContentId())
	}

	var contentCount int64
	if err := db.Model(&model.ZfeedContent{}).Where("id = ?", resp.GetContentId()).Count(&contentCount).Error; err != nil {
		t.Fatalf("count zfeed_content: %v", err)
	}
	if contentCount != 1 {
		t.Fatalf("content row count = %d, want 1", contentCount)
	}

	var articleCount int64
	if err := db.Model(&model.ZfeedArticle{}).Where("content_id = ?", resp.GetContentId()).Count(&articleCount).Error; err != nil {
		t.Fatalf("count zfeed_article: %v", err)
	}
	if articleCount != 1 {
		t.Fatalf("article row count = %d, want 1", articleCount)
	}
}
