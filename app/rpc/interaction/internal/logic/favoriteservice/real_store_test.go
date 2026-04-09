package favoriteservicelogic

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	followservicelogic "zfeed/app/rpc/interaction/internal/logic/followservice"
	interactionmodel "zfeed/app/rpc/interaction/internal/model"
	interactionsvc "zfeed/app/rpc/interaction/internal/svc"
	"zfeed/app/rpc/interaction/internal/testutil/mysqltest"
)

const runRealStoreEnv = "RUN_REAL_STORE"

type realStoreContentServiceAdapter struct {
	redisClient *gzredis.Redis
}

var _ contentservice.ContentService = (*realStoreContentServiceAdapter)(nil)

func (a *realStoreContentServiceAdapter) PublishArticle(ctx context.Context, in *contentpb.ArticlePublishReq, opts ...grpc.CallOption) (*contentpb.ArticlePublishRes, error) {
	return nil, grpc.ErrClientConnClosing
}

func (a *realStoreContentServiceAdapter) PublishVideo(ctx context.Context, in *contentpb.VideoPublishReq, opts ...grpc.CallOption) (*contentpb.VideoPublishRes, error) {
	return nil, grpc.ErrClientConnClosing
}

func (a *realStoreContentServiceAdapter) BackfillFollowInbox(ctx context.Context, in *contentpb.BackfillFollowInboxReq, opts ...grpc.CallOption) (*contentpb.BackfillFollowInboxRes, error) {
	publishKey := "feed:user:publish:" + int64ToString(in.GetFolloweeId())
	inboxKey := "feed:follow:inbox:" + int64ToString(in.GetFollowerId())

	members, err := a.redisClient.ZrevrangeCtx(ctx, publishKey, 0, int64(in.GetLimit())-1)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		score, convErr := strconv.ParseInt(member, 10, 64)
		if convErr != nil {
			return nil, convErr
		}
		if _, err := a.redisClient.ZaddCtx(ctx, inboxKey, score, member); err != nil {
			return nil, err
		}
	}

	return &contentpb.BackfillFollowInboxRes{AddedCount: int32(len(members))}, nil
}

func TestRealStoreFavoriteAndFollowFlow(t *testing.T) {
	if os.Getenv(runRealStoreEnv) != "1" {
		t.Skipf("set %s=1 to run real MySQL/Redis verification", runRealStoreEnv)
	}

	db, err := mysqltest.Open()
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer func() {
		_ = mysqltest.Close(db)
	}()

	if err := db.AutoMigrate(
		&interactionmodel.ZfeedFavorite{},
		&interactionmodel.ZfeedFollow{},
	); err != nil {
		t.Fatalf("auto migrate interaction models: %v", err)
	}
	if err := ensureRealStoreContentTables(db); err != nil {
		t.Fatalf("ensure content tables: %v", err)
	}

	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: getenvDefault("REDIS_ADDR", "127.0.0.1:16379"),
		Type: "node",
	})

	cleanupRealStoreTables(t, db)

	ctx := context.Background()
	interactionSvcCtx := &interactionsvc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		ContentRpc: &realStoreContentServiceAdapter{
			redisClient: redisClient,
		},
	}

	var (
		viewerID   int64 = 101001
		followeeID int64 = 202002
	)

	favoriteFeedKey := rediskey.BuildUserFavoriteFeedKey(int64ToString(viewerID))
	if _, err := redisClient.ZaddCtx(ctx, favoriteFeedKey, 1, "seed"); err != nil {
		t.Fatalf("seed favorite feed cache: %v", err)
	}

	firstContent, secondContent := seedFolloweePublishZSet(t, redisClient, followeeID)
	t.Logf("seeded publish zset content ids: %d, %d", firstContent, secondContent)

	favoriteLogic := NewFavoriteLogic(ctx, interactionSvcCtx)
	removeFavoriteLogic := NewRemoveFavoriteLogic(ctx, interactionSvcCtx)
	queryFavoriteInfoLogic := NewQueryFavoriteInfoLogic(ctx, interactionSvcCtx)

	favoriteReq := &interaction.FavoriteReq{
		UserId:        viewerID,
		ContentId:     firstContent,
		ContentUserId: followeeID,
		Scene:         interaction.Scene_ARTICLE,
	}
	if _, err := favoriteLogic.Favorite(favoriteReq); err != nil {
		t.Fatalf("favorite once: %v", err)
	}
	if _, err := favoriteLogic.Favorite(favoriteReq); err != nil {
		t.Fatalf("favorite twice: %v", err)
	}

	relKey := rediskey.BuildFavoriteRelKey(interaction.Scene_ARTICLE.String(), int64ToString(viewerID), int64ToString(firstContent))
	existsAfterFavorite, err := redisClient.ExistsCtx(ctx, relKey)
	if err != nil {
		t.Fatalf("exists favorite rel after favorite: %v", err)
	}
	if existsAfterFavorite {
		t.Fatalf("favorite rel cache should be deleted immediately after favorite write")
	}

	favoriteInfo, err := queryFavoriteInfoLogic.QueryFavoriteInfo(&interaction.QueryFavoriteInfoReq{
		UserId:    viewerID,
		ContentId: firstContent,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("query favorite info after favorite: %v", err)
	}
	if !favoriteInfo.GetIsFavorited() || favoriteInfo.GetFavoriteCount() != 1 {
		t.Fatalf("favorite info after favorite = %+v, want count=1 and is_favorited=true", favoriteInfo)
	}

	relVal, err := redisClient.GetCtx(ctx, relKey)
	if err != nil {
		t.Fatalf("get favorite rel cache after query: %v", err)
	}

	favoriteScore, err := redisClient.ZscoreCtx(ctx, favoriteFeedKey, int64ToString(firstContent))
	if err != nil {
		t.Fatalf("zscore favorite feed: %v", err)
	}

	var favoriteRows int64
	if err := db.Table("zfeed_favorite").Where("user_id = ? AND content_id = ?", viewerID, firstContent).Count(&favoriteRows).Error; err != nil {
		t.Fatalf("count favorite rows: %v", err)
	}

	t.Logf("favorite state after idempotent favorite: rows=%d rel=%s feed_score=%d", favoriteRows, relVal, favoriteScore)

	removeReq := &interaction.RemoveFavoriteReq{
		UserId:    viewerID,
		ContentId: firstContent,
		Scene:     interaction.Scene_ARTICLE,
	}
	if _, err := removeFavoriteLogic.RemoveFavorite(removeReq); err != nil {
		t.Fatalf("remove favorite once: %v", err)
	}
	if _, err := removeFavoriteLogic.RemoveFavorite(removeReq); err != nil {
		t.Fatalf("remove favorite twice: %v", err)
	}

	favoriteInfo, err = queryFavoriteInfoLogic.QueryFavoriteInfo(&interaction.QueryFavoriteInfoReq{
		UserId:    viewerID,
		ContentId: firstContent,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("query favorite info after remove: %v", err)
	}
	if favoriteInfo.GetIsFavorited() || favoriteInfo.GetFavoriteCount() != 0 {
		t.Fatalf("favorite info after remove = %+v, want count=0 and is_favorited=false", favoriteInfo)
	}

	relVal, err = redisClient.GetCtx(ctx, relKey)
	if err != nil {
		t.Fatalf("get favorite rel cache after remove query: %v", err)
	}

	favoriteMembers, err := redisClient.ZrangeCtx(ctx, favoriteFeedKey, 0, -1)
	if err != nil {
		t.Fatalf("zrange favorite feed after remove: %v", err)
	}
	t.Logf("favorite state after idempotent unfavorite: rel=%s feed_members=%v", relVal, favoriteMembers)

	followLogic := followservicelogic.NewFollowUserLogic(ctx, interactionSvcCtx)
	unfollowLogic := followservicelogic.NewUnfollowUserLogic(ctx, interactionSvcCtx)
	listFolloweesLogic := followservicelogic.NewListFolloweesLogic(ctx, interactionSvcCtx)
	getFollowSummaryLogic := followservicelogic.NewGetFollowSummaryLogic(ctx, interactionSvcCtx)

	followReq := &interaction.FollowUserReq{
		UserId:       viewerID,
		FollowUserId: followeeID,
	}
	if _, err := followLogic.FollowUser(followReq); err != nil {
		t.Fatalf("follow once: %v", err)
	}
	if _, err := followLogic.FollowUser(followReq); err != nil {
		t.Fatalf("follow twice: %v", err)
	}

	inboxKey := "feed:follow:inbox:" + int64ToString(viewerID)
	inboxMembers := waitForInboxMembers(t, redisClient, inboxKey, 2)

	listResp, err := listFolloweesLogic.ListFollowees(&interaction.ListFolloweesReq{
		UserId:   viewerID,
		Cursor:   0,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list followees after follow: %v", err)
	}

	summaryResp, err := getFollowSummaryLogic.GetFollowSummary(&interaction.GetFollowSummaryReq{
		UserId:   followeeID,
		ViewerId: &viewerID,
	})
	if err != nil {
		t.Fatalf("follow summary after follow: %v", err)
	}

	var followRows int64
	var followStatus int32
	if err := db.Table("zfeed_follow").Where("user_id = ? AND follow_user_id = ?", viewerID, followeeID).Count(&followRows).Error; err != nil {
		t.Fatalf("count follow rows: %v", err)
	}
	if err := db.Table("zfeed_follow").Select("status").Where("user_id = ? AND follow_user_id = ?", viewerID, followeeID).Take(&followStatus).Error; err != nil {
		t.Fatalf("query follow status after follow: %v", err)
	}

	t.Logf("follow state after idempotent follow: rows=%d status=%d followees=%v inbox_members=%v follower_count=%d is_following=%v",
		followRows, followStatus, listResp.GetFollowUserIds(), inboxMembers, summaryResp.GetFollowerCount(), summaryResp.GetIsFollowing())

	unfollowReq := &interaction.UnfollowUserReq{
		UserId:       viewerID,
		FollowUserId: followeeID,
	}
	if _, err := unfollowLogic.UnfollowUser(unfollowReq); err != nil {
		t.Fatalf("unfollow once: %v", err)
	}
	if _, err := unfollowLogic.UnfollowUser(unfollowReq); err != nil {
		t.Fatalf("unfollow twice: %v", err)
	}

	if err := db.Table("zfeed_follow").Select("status").Where("user_id = ? AND follow_user_id = ?", viewerID, followeeID).Take(&followStatus).Error; err != nil {
		t.Fatalf("query follow status after unfollow: %v", err)
	}

	summaryResp, err = getFollowSummaryLogic.GetFollowSummary(&interaction.GetFollowSummaryReq{
		UserId:   followeeID,
		ViewerId: &viewerID,
	})
	if err != nil {
		t.Fatalf("follow summary after unfollow: %v", err)
	}

	t.Logf("follow state after idempotent unfollow: status=%d inbox_members=%v follower_count=%d is_following=%v",
		followStatus, inboxMembers, summaryResp.GetFollowerCount(), summaryResp.GetIsFollowing())
}

func seedFolloweePublishZSet(t *testing.T, redisClient *gzredis.Redis, followeeID int64) (int64, int64) {
	t.Helper()

	publishKey := "feed:user:publish:" + int64ToString(followeeID)
	firstContent := int64(900001)
	secondContent := int64(900002)

	if _, err := redisClient.ZaddCtx(context.Background(), publishKey, firstContent, int64ToString(firstContent)); err != nil {
		t.Fatalf("seed first publish zset member: %v", err)
	}
	if _, err := redisClient.ZaddCtx(context.Background(), publishKey, secondContent, int64ToString(secondContent)); err != nil {
		t.Fatalf("seed second publish zset member: %v", err)
	}

	return firstContent, secondContent
}

func waitForInboxMembers(t *testing.T, redisClient *gzredis.Redis, key string, want int) []string {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		members, err := redisClient.ZrangeCtx(context.Background(), key, 0, -1)
		if err == nil && len(members) >= want {
			return members
		}
		time.Sleep(50 * time.Millisecond)
	}

	members, err := redisClient.ZrangeCtx(context.Background(), key, 0, -1)
	if err != nil {
		t.Fatalf("query inbox members: %v", err)
	}
	t.Fatalf("inbox members = %v, want at least %d", members, want)
	return nil
}

func cleanupRealStoreTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	for _, stmt := range []string{
		"DELETE FROM zfeed_follow",
		"DELETE FROM zfeed_favorite",
		"DELETE FROM zfeed_article",
		"DELETE FROM zfeed_video",
		"DELETE FROM zfeed_content",
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("cleanup table with %q: %v", stmt, err)
		}
	}
}

func ensureRealStoreContentTables(db *gorm.DB) error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS zfeed_content (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL DEFAULT 0,
			content_type INT NOT NULL DEFAULT 0,
			status INT NOT NULL DEFAULT 0,
			visibility INT NOT NULL DEFAULT 0,
			like_count BIGINT NOT NULL DEFAULT 0,
			favorite_count BIGINT NOT NULL DEFAULT 0,
			comment_count BIGINT NOT NULL DEFAULT 0,
			published_at DATETIME DEFAULT NULL,
			is_deleted TINYINT NOT NULL DEFAULT 0,
			created_by BIGINT NOT NULL DEFAULT 0,
			updated_by BIGINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS zfeed_article (
			id BIGINT NOT NULL AUTO_INCREMENT,
			content_id BIGINT NOT NULL DEFAULT 0,
			title VARCHAR(255) NOT NULL DEFAULT '',
			description TEXT NULL,
			cover VARCHAR(1024) NOT NULL DEFAULT '',
			content LONGTEXT NOT NULL,
			is_deleted TINYINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS zfeed_video (
			id BIGINT NOT NULL AUTO_INCREMENT,
			content_id BIGINT NOT NULL DEFAULT 0,
			title VARCHAR(255) NOT NULL DEFAULT '',
			description TEXT NULL,
			origin_url VARCHAR(1024) NOT NULL DEFAULT '',
			cover_url VARCHAR(1024) NOT NULL DEFAULT '',
			duration INT NOT NULL DEFAULT 0,
			transcode_status INT NOT NULL DEFAULT 0,
			is_deleted TINYINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func getenvDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
