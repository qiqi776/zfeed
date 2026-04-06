package favoriteservicelogic

import (
	"context"
	"strconv"
	"strings"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/model"
	"zfeed/app/rpc/interaction/internal/svc"
)

func newFavoriteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedFavorite{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func newFavoriteTestRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	return store, client
}

func TestFavoriteAndRemoveFavorite_UpdateDBAndCache(t *testing.T) {
	db := newFavoriteTestDB(t)
	store, client := newFavoriteTestRedis(t)
	defer store.Close()

	logicCtx := &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	}

	favoriteLogic := NewFavoriteLogic(context.Background(), logicCtx)
	removeLogic := NewRemoveFavoriteLogic(context.Background(), logicCtx)
	queryLogic := NewQueryFavoriteInfoLogic(context.Background(), logicCtx)

	const (
		userID        int64 = 1001
		contentID     int64 = 2002
		contentUserID int64 = 3003
	)

	relKey := rediskey.BuildFavoriteRelKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(userID, 10), strconv.FormatInt(contentID, 10))
	favoriteFeedKey := rediskey.BuildUserFavoriteFeedKey(strconv.FormatInt(userID, 10))

	store.Set(relKey, "stale")
	store.ZAdd(favoriteFeedKey, 1, "seed")

	_, err := favoriteLogic.Favorite(&interaction.FavoriteReq{
		UserId:        userID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("Favorite returned error: %v", err)
	}

	if store.Exists(relKey) {
		t.Fatalf("relation cache key %q still exists after favorite", relKey)
	}

	var row model.ZfeedFavorite
	if err := db.Where("user_id = ? AND content_id = ?", userID, contentID).Take(&row).Error; err != nil {
		t.Fatalf("query favorite row: %v", err)
	}

	score, err := store.ZScore(favoriteFeedKey, strconv.FormatInt(contentID, 10))
	if err != nil {
		t.Fatalf("query favorite feed score: %v", err)
	}
	if int64(score) != row.ID {
		t.Fatalf("favorite feed score = %v, want %d", score, row.ID)
	}

	infoResp, err := queryLogic.QueryFavoriteInfo(&interaction.QueryFavoriteInfoReq{
		UserId:    userID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("QueryFavoriteInfo after favorite returned error: %v", err)
	}
	if !infoResp.GetIsFavorited() || infoResp.GetFavoriteCount() != 1 {
		t.Fatalf("favorite info after favorite = %+v, want is_favorited=true and count=1", infoResp)
	}
	if value, err := store.Get(relKey); err != nil || value != "1" {
		t.Fatalf("relation cache after favorite = (%q, %v), want (\"1\", nil)", value, err)
	}

	_, err = removeLogic.RemoveFavorite(&interaction.RemoveFavoriteReq{
		UserId:    userID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("RemoveFavorite returned error: %v", err)
	}

	if store.Exists(relKey) {
		t.Fatalf("relation cache key %q still exists after remove favorite", relKey)
	}
	members, err := store.ZMembers(favoriteFeedKey)
	if err != nil {
		t.Fatalf("query favorite feed members after remove: %v", err)
	}
	for _, member := range members {
		if member == strconv.FormatInt(contentID, 10) {
			t.Fatal("favorite feed member still exists after remove favorite")
		}
	}

	var count int64
	if err := db.Model(&model.ZfeedFavorite{}).Where("user_id = ? AND content_id = ?", userID, contentID).Count(&count).Error; err != nil {
		t.Fatalf("count favorite rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("favorite row count = %d, want 0", count)
	}

	infoResp, err = queryLogic.QueryFavoriteInfo(&interaction.QueryFavoriteInfoReq{
		UserId:    userID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("QueryFavoriteInfo after remove returned error: %v", err)
	}
	if infoResp.GetIsFavorited() || infoResp.GetFavoriteCount() != 0 {
		t.Fatalf("favorite info after remove = %+v, want is_favorited=false and count=0", infoResp)
	}
	if value, err := store.Get(relKey); err != nil || value != "0" {
		t.Fatalf("relation cache after remove = (%q, %v), want (\"0\", nil)", value, err)
	}
}
