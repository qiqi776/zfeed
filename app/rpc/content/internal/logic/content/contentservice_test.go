package contentlogic

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/interaction/client/likeservice"
)

type contentServiceTestContent struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	UserID        int64      `gorm:"column:user_id"`
	ContentType   int32      `gorm:"column:content_type"`
	Status        int32      `gorm:"column:status"`
	Visibility    int32      `gorm:"column:visibility"`
	LikeCount     int64      `gorm:"column:like_count"`
	FavoriteCount int64      `gorm:"column:favorite_count"`
	CommentCount  int64      `gorm:"column:comment_count"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	IsDeleted     int32      `gorm:"column:is_deleted"`
	UpdatedBy     int64      `gorm:"column:updated_by"`
}

func (contentServiceTestContent) TableName() string { return "zfeed_content" }

type contentServiceTestArticle struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	Cover       string  `gorm:"column:cover"`
	Content     string  `gorm:"column:content"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (contentServiceTestArticle) TableName() string { return "zfeed_article" }

type contentServiceTestVideo struct {
	ContentID   int64   `gorm:"column:content_id;primaryKey"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	OriginURL   string  `gorm:"column:origin_url"`
	CoverURL    string  `gorm:"column:cover_url"`
	Duration    int32   `gorm:"column:duration"`
	IsDeleted   int32   `gorm:"column:is_deleted"`
}

func (contentServiceTestVideo) TableName() string { return "zfeed_video" }

type contentServiceTestUser struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Nickname  string `gorm:"column:nickname"`
	Avatar    string `gorm:"column:avatar"`
	IsDeleted int32  `gorm:"column:is_deleted"`
}

func (contentServiceTestUser) TableName() string { return "zfeed_user" }

type contentServiceTestLike struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64 `gorm:"column:user_id"`
	ContentID int64 `gorm:"column:content_id"`
	Status    int32 `gorm:"column:status"`
	IsDeleted int32 `gorm:"column:is_deleted"`
}

func (contentServiceTestLike) TableName() string { return "zfeed_like" }

type contentServiceTestFavorite struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64 `gorm:"column:user_id"`
	ContentID int64 `gorm:"column:content_id"`
	Status    int32 `gorm:"column:status"`
}

func (contentServiceTestFavorite) TableName() string { return "zfeed_favorite" }

type contentServiceTestFollow struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64 `gorm:"column:user_id"`
	FollowUserID int64 `gorm:"column:follow_user_id"`
	Status       int32 `gorm:"column:status"`
	IsDeleted    int32 `gorm:"column:is_deleted"`
}

func (contentServiceTestFollow) TableName() string { return "zfeed_follow" }

type stubLikeService struct {
	likeservice.LikeService
	queryLikeInfo func(context.Context, *likeservice.QueryLikeInfoReq, ...grpc.CallOption) (*likeservice.QueryLikeInfoRes, error)
}

func (s *stubLikeService) QueryLikeInfo(ctx context.Context, in *likeservice.QueryLikeInfoReq, opts ...grpc.CallOption) (*likeservice.QueryLikeInfoRes, error) {
	if s.queryLikeInfo == nil {
		return nil, errors.New("unexpected QueryLikeInfo call")
	}
	return s.queryLikeInfo(ctx, in, opts...)
}

type stubFavoriteService struct {
	favoriteservice.FavoriteService
	queryFavoriteInfo func(context.Context, *favoriteservice.QueryFavoriteInfoReq, ...grpc.CallOption) (*favoriteservice.QueryFavoriteInfoRes, error)
}

func (s *stubFavoriteService) QueryFavoriteInfo(ctx context.Context, in *favoriteservice.QueryFavoriteInfoReq, opts ...grpc.CallOption) (*favoriteservice.QueryFavoriteInfoRes, error) {
	if s.queryFavoriteInfo == nil {
		return nil, errors.New("unexpected QueryFavoriteInfo call")
	}
	return s.queryFavoriteInfo(ctx, in, opts...)
}

type stubFollowService struct {
	followservice.FollowService
	batchQueryFollowing func(context.Context, *followservice.BatchQueryFollowingReq, ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error)
}

func (s *stubFollowService) BatchQueryFollowing(ctx context.Context, in *followservice.BatchQueryFollowingReq, opts ...grpc.CallOption) (*followservice.BatchQueryFollowingRes, error) {
	if s.batchQueryFollowing == nil {
		return nil, errors.New("unexpected BatchQueryFollowing call")
	}
	return s.batchQueryFollowing(ctx, in, opts...)
}

func newContentServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&contentServiceTestContent{},
		&contentServiceTestArticle{},
		&contentServiceTestVideo{},
		&contentServiceTestUser{},
		&contentServiceTestLike{},
		&contentServiceTestFavorite{},
		&contentServiceTestFollow{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestPrivateAuthor(t *testing.T) {
	db := newContentServiceTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&contentServiceTestUser{ID: 1001, Nickname: "author", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&contentServiceTestContent{
		ID:          5001,
		UserID:      1001,
		ContentType: contentTypeArticle,
		Status:      contentStatusPublish,
		Visibility:  contentVisibilityPrivate,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{
		ContentID: 5001,
		Title:     "private article",
		Cover:     "cover",
		Content:   "body",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewGetContentDetailLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.GetContentDetail(&content.GetContentDetailReq{
		ContentId: 5001,
		ViewerId:  int64Ptr(1001),
	})
	if err != nil {
		t.Fatalf("GetContentDetail returned error: %v", err)
	}
	if resp.GetDetail() == nil || resp.GetDetail().GetContentId() != 5001 || resp.GetDetail().GetTitle() != "private article" {
		t.Fatalf("unexpected detail: %+v", resp.GetDetail())
	}
}

func TestPrivateDenied(t *testing.T) {
	db := newContentServiceTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&contentServiceTestUser{ID: 1001, Nickname: "author", Avatar: "avatar", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&contentServiceTestContent{
		ID:          5002,
		UserID:      1001,
		ContentType: contentTypeArticle,
		Status:      contentStatusPublish,
		Visibility:  contentVisibilityPrivate,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{
		ContentID: 5002,
		Title:     "private article",
		Cover:     "cover",
		Content:   "body",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewGetContentDetailLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	if _, err := logic.GetContentDetail(&content.GetContentDetailReq{
		ContentId: 5002,
		ViewerId:  int64Ptr(2002),
	}); err == nil {
		t.Fatal("expected private content to be hidden from other viewers")
	}
}

func TestEditArticle(t *testing.T) {
	db := newContentServiceTestDB(t)
	if err := db.Create(&contentServiceTestContent{ID: 101, UserID: 1, ContentType: 10, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{ContentID: 101, Title: "old", Cover: "old-cover", Content: "old body", IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewEditArticleLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.EditArticle(&content.EditArticleReq{
		UserId:    1,
		ContentId: 101,
		Title:     stringPtr("new-title"),
		Content:   stringPtr("new body"),
	})
	if err != nil {
		t.Fatalf("EditArticle returned error: %v", err)
	}
	if resp.GetContentId() != 101 {
		t.Fatalf("content_id = %d, want 101", resp.GetContentId())
	}

	var row contentServiceTestArticle
	if err := db.Table("zfeed_article").Where("content_id = ?", 101).Take(&row).Error; err != nil {
		t.Fatalf("query article: %v", err)
	}
	if row.Title != "new-title" || row.Content != "new body" {
		t.Fatalf("unexpected article row: %+v", row)
	}
}

func TestEditVideo(t *testing.T) {
	db := newContentServiceTestDB(t)
	if err := db.Create(&contentServiceTestContent{ID: 202, UserID: 2, ContentType: 20, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestVideo{ContentID: 202, Title: "old", OriginURL: "old-url", CoverURL: "old-cover", Duration: 12, IsDeleted: 0}).Error; err != nil {
		t.Fatalf("seed video: %v", err)
	}

	logic := NewEditVideoLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.EditVideo(&content.EditVideoReq{
		UserId:    2,
		ContentId: 202,
		Title:     stringPtr("new-video"),
		OriginUrl: stringPtr("https://example.com/new.mp4"),
		Duration:  int32Ptr(66),
	})
	if err != nil {
		t.Fatalf("EditVideo returned error: %v", err)
	}
	if resp.GetContentId() != 202 {
		t.Fatalf("content_id = %d, want 202", resp.GetContentId())
	}

	var row contentServiceTestVideo
	if err := db.Table("zfeed_video").Where("content_id = ?", 202).Take(&row).Error; err != nil {
		t.Fatalf("query video: %v", err)
	}
	if row.Title != "new-video" || row.OriginURL != "https://example.com/new.mp4" || row.Duration != 66 {
		t.Fatalf("unexpected video row: %+v", row)
	}
}

func TestDelete(t *testing.T) {
	db := newContentServiceTestDB(t)
	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	if err := db.Create(&contentServiceTestContent{
		ID:          301,
		UserID:      7,
		ContentType: 10,
		Status:      30,
		Visibility:  10,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{
		ContentID: 301,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}
	if err := db.Create(&contentServiceTestFollow{
		UserID:       41,
		FollowUserID: 7,
		Status:       10,
		IsDeleted:    0,
	}).Error; err != nil {
		t.Fatalf("seed follower 41: %v", err)
	}
	if err := db.Create(&contentServiceTestFollow{
		UserID:       42,
		FollowUserID: 7,
		Status:       10,
		IsDeleted:    0,
	}).Error; err != nil {
		t.Fatalf("seed follower 42: %v", err)
	}
	if err := db.Create(&contentServiceTestFavorite{
		UserID:    51,
		ContentID: 301,
		Status:    10,
	}).Error; err != nil {
		t.Fatalf("seed favorite 51: %v", err)
	}
	if err := db.Create(&contentServiceTestFavorite{
		UserID:    52,
		ContentID: 301,
		Status:    20,
	}).Error; err != nil {
		t.Fatalf("seed favorite 52: %v", err)
	}

	contentID := strconv.FormatInt(301, 10)
	publishKey := redisconsts.BuildUserPublishFeedKey(7)
	store.ZAdd(publishKey, 301, contentID)
	store.ZAdd(publishKey, 999, "999")
	follower41InboxKey := redisconsts.BuildFollowInboxKey(41)
	store.ZAdd(follower41InboxKey, 301, contentID)
	store.ZAdd(follower41InboxKey, 999, "999")
	follower42InboxKey := redisconsts.BuildFollowInboxKey(42)
	store.ZAdd(follower42InboxKey, 301, contentID)
	favorite51Key := redisconsts.BuildUserFavoriteFeedKey(51)
	store.ZAdd(favorite51Key, 301, contentID)
	store.ZAdd(favorite51Key, 999, "999")
	favorite52Key := redisconsts.BuildUserFavoriteFeedKey(52)
	store.ZAdd(favorite52Key, 301, contentID)
	store.ZAdd(redisconsts.HotFeedKey, 301, contentID)
	store.ZAdd(redisconsts.HotFeedKey, 999, "999")
	store.ZAdd(redisconsts.RecommendNewContentKey, 301, contentID)
	store.ZAdd(redisconsts.RecommendNewContentKey, 999, "999")
	if err := redisClient.HsetCtx(context.Background(), redisconsts.BuildRecommendNewContentMetaKey(301), "author_id", "7"); err != nil {
		t.Fatalf("seed new content meta: %v", err)
	}
	contentTagsKey := redisconsts.BuildRecommendContentTagsKey(301)
	if err := redisClient.HsetCtx(context.Background(), contentTagsKey, "go", "0.8"); err != nil {
		t.Fatalf("seed content tag go: %v", err)
	}
	if err := redisClient.HsetCtx(context.Background(), contentTagsKey, "type:article", "1"); err != nil {
		t.Fatalf("seed content tag type:article: %v", err)
	}
	tagGoKey := redisconsts.BuildRecommendTagIndexKey("go")
	tagArticleKey := redisconsts.BuildRecommendTagIndexKey("type:article")
	store.ZAdd(tagGoKey, 301, contentID)
	store.ZAdd(tagGoKey, 999, "999")
	store.ZAdd(tagArticleKey, 301, contentID)
	snapshotID := "snap-delete"
	store.Set(redisconsts.HotFeedLatestKey, snapshotID)
	snapshotKey := redisconsts.BuildHotFeedSnapshotKey(snapshotID)
	store.ZAdd(snapshotKey, 301, contentID)
	store.ZAdd(snapshotKey, 999, "999")
	incKey := redisconsts.BuildHotFeedIncKey(int(301 % int64(redisconsts.HotFeedIncShards)))
	if err := redisClient.HsetCtx(context.Background(), incKey, contentID, "1"); err != nil {
		t.Fatalf("seed hot inc content: %v", err)
	}
	if err := redisClient.HsetCtx(context.Background(), incKey, "999", "1"); err != nil {
		t.Fatalf("seed hot inc other: %v", err)
	}

	logic := NewDeleteContentLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})
	if _, err := logic.DeleteContent(&content.DeleteContentReq{UserId: 7, ContentId: 301}); err != nil {
		t.Fatalf("DeleteContent returned error: %v", err)
	}

	var contentRow contentServiceTestContent
	if err := db.Table("zfeed_content").Where("id = ?", 301).Take(&contentRow).Error; err != nil {
		t.Fatalf("query deleted content: %v", err)
	}
	if contentRow.IsDeleted != 1 || contentRow.UpdatedBy != 7 {
		t.Fatalf("unexpected deleted content row: %+v", contentRow)
	}

	var articleRow contentServiceTestArticle
	if err := db.Table("zfeed_article").Where("content_id = ?", 301).Take(&articleRow).Error; err != nil {
		t.Fatalf("query deleted article: %v", err)
	}
	if articleRow.IsDeleted != 1 {
		t.Fatalf("article is_deleted = %d, want 1", articleRow.IsDeleted)
	}
	assertZSetMissing(t, store, publishKey, contentID)
	assertZSetMissing(t, store, follower41InboxKey, contentID)
	assertZSetMissing(t, store, follower42InboxKey, contentID)
	assertZSetMissing(t, store, favorite51Key, contentID)
	assertZSetMissing(t, store, favorite52Key, contentID)
	assertZSetMissing(t, store, redisconsts.HotFeedKey, contentID)
	assertZSetMissing(t, store, redisconsts.RecommendNewContentKey, contentID)
	assertZSetMissing(t, store, snapshotKey, contentID)
	if store.Exists(redisconsts.BuildRecommendNewContentMetaKey(301)) {
		t.Fatalf("new content meta still exists after delete")
	}
	if store.Exists(contentTagsKey) {
		t.Fatalf("content tags still exist after delete")
	}
	assertZSetMissing(t, store, tagGoKey, contentID)
	assertZSetMissing(t, store, tagArticleKey, contentID)

	incMap, err := redisClient.HgetallCtx(context.Background(), incKey)
	if err != nil {
		t.Fatalf("read hot inc bucket: %v", err)
	}
	if _, ok := incMap[contentID]; ok {
		t.Fatalf("hot inc bucket still has deleted content: %v", incMap)
	}
	if incMap["999"] != "1" {
		t.Fatalf("hot inc bucket removed unrelated member: %v", incMap)
	}
}

func TestDeleteOther(t *testing.T) {
	db := newContentServiceTestDB(t)
	if err := db.Create(&contentServiceTestContent{
		ID:          302,
		UserID:      9,
		ContentType: 20,
		Status:      30,
		Visibility:  10,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestVideo{
		ContentID: 302,
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed video: %v", err)
	}

	logic := NewDeleteContentLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	if _, err := logic.DeleteContent(&content.DeleteContentReq{UserId: 7, ContentId: 302}); err == nil {
		t.Fatal("expected forbidden delete")
	}

	var contentRow contentServiceTestContent
	if err := db.Table("zfeed_content").Where("id = ?", 302).Take(&contentRow).Error; err != nil {
		t.Fatalf("query content after forbidden delete: %v", err)
	}
	if contentRow.IsDeleted != 0 {
		t.Fatalf("content is_deleted = %d, want 0", contentRow.IsDeleted)
	}
}

func int64Ptr(value int64) *int64 { return &value }

func int32Ptr(value int32) *int32 { return &value }

func stringPtr(value string) *string { return &value }

func assertZSetMissing(t *testing.T, store *miniredis.Miniredis, key string, member string) {
	t.Helper()

	if !store.Exists(key) {
		return
	}
	members, err := store.ZMembers(key)
	if err != nil {
		t.Fatalf("zset %s members: %v", key, err)
	}
	for _, value := range members {
		if value == member {
			t.Fatalf("zset %s still has member %s: %v", key, member, members)
		}
	}
}
