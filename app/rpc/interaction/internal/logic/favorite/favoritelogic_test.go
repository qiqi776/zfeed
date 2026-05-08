package favoritelogic

import (
	"context"
	"strconv"
	"strings"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/logx"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/model"
	"zfeed/app/rpc/interaction/internal/repositories"
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
	if err := db.AutoMigrate(&model.ZfeedFavoriteEvent{}); err != nil {
		t.Fatalf("auto migrate favorite events: %v", err)
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

type favoriteStubContentRepo struct {
	getAuthorIDFunc func(contentID int64) (int64, error)
}

func (r *favoriteStubContentRepo) GetAuthorID(contentID int64) (int64, error) {
	if r.getAuthorIDFunc == nil {
		return 0, nil
	}
	return r.getAuthorIDFunc(contentID)
}

type favoriteStubCommentRepo struct {
	getByIDFunc func(commentID int64) (*do.CommentDO, error)
}

func (r *favoriteStubCommentRepo) WithTx(tx *gorm.DB) repositories.CommentRepository { return r }
func (r *favoriteStubCommentRepo) Create(*do.CommentDO) (int64, error)               { return 0, nil }
func (r *favoriteStubCommentRepo) GetByID(commentID int64) (*do.CommentDO, error) {
	if r.getByIDFunc == nil {
		return nil, nil
	}
	return r.getByIDFunc(commentID)
}
func (r *favoriteStubCommentRepo) GetByIDIncludeDeleted(int64) (*do.CommentDO, error) {
	return nil, nil
}
func (r *favoriteStubCommentRepo) ListRootComments(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, nil
}
func (r *favoriteStubCommentRepo) ListRootCommentsIncludeDeleted(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, nil
}
func (r *favoriteStubCommentRepo) ListReplies(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, nil
}
func (r *favoriteStubCommentRepo) ListRepliesIncludeDeleted(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, nil
}
func (r *favoriteStubCommentRepo) BatchGetByIDs([]int64) (map[int64]*do.CommentDO, error) {
	return nil, nil
}
func (r *favoriteStubCommentRepo) BatchGetByIDsIncludeDeleted([]int64) (map[int64]*do.CommentDO, error) {
	return nil, nil
}
func (r *favoriteStubCommentRepo) MarkDeleted(int64, int64) error  { return nil }
func (r *favoriteStubCommentRepo) DeleteByID(int64) error          { return nil }
func (r *favoriteStubCommentRepo) HasChildren(int64) (bool, error) { return false, nil }
func (r *favoriteStubCommentRepo) IncReplyCount(int64) error       { return nil }
func (r *favoriteStubCommentRepo) DecReplyCount(int64) error       { return nil }

func TestFavoriteAndRemoveFavorite_UpdateDBAndCache(t *testing.T) {
	db := newFavoriteTestDB(t)
	store, client := newFavoriteTestRedis(t)
	defer store.Close()

	logicCtx := &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	}
	queryLogic := NewQueryFavoriteInfoLogic(context.Background(), logicCtx)

	const (
		userID        int64 = 1001
		contentID     int64 = 2002
		contentUserID int64 = 3003
	)

	favoriteLogic := &FavoriteLogic{
		ctx:               context.Background(),
		svcCtx:            logicCtx,
		Logger:            logx.WithContext(context.Background()),
		favoriteRepo:      repositories.NewFavoriteRepository(context.Background(), db),
		favoriteEventRepo: repositories.NewFavoriteEventRepository(context.Background(), db),
		contentRepo: &favoriteStubContentRepo{
			getAuthorIDFunc: func(gotContentID int64) (int64, error) {
				if gotContentID != contentID {
					t.Fatalf("unexpected content_id=%d", gotContentID)
				}
				return contentUserID, nil
			},
		},
		commentRepo: &favoriteStubCommentRepo{},
	}
	removeLogic := &RemoveFavoriteLogic{
		ctx:               context.Background(),
		svcCtx:            logicCtx,
		Logger:            logx.WithContext(context.Background()),
		favoriteRepo:      repositories.NewFavoriteRepository(context.Background(), db),
		favoriteEventRepo: repositories.NewFavoriteEventRepository(context.Background(), db),
		contentRepo: &favoriteStubContentRepo{
			getAuthorIDFunc: func(gotContentID int64) (int64, error) {
				if gotContentID != contentID {
					t.Fatalf("unexpected content_id=%d", gotContentID)
				}
				return contentUserID, nil
			},
		},
		commentRepo: &favoriteStubCommentRepo{},
	}

	relKey := rediskey.BuildFavoriteRelKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(userID, 10), strconv.FormatInt(contentID, 10))
	favoriteFeedKey := rediskey.BuildUserFavoriteFeedKey(strconv.FormatInt(userID, 10))

	store.Set(relKey, "stale")
	store.ZAdd(favoriteFeedKey, 1, "seed")

	_, err := favoriteLogic.Favorite(&interaction.FavoriteReq{
		UserId:    userID,
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("Favorite returned error: %v", err)
	}

	if store.Exists(relKey) {
		t.Fatalf("relation cache key %q still exists after favorite", relKey)
	}

	var row model.ZfeedFavorite
	if err := db.Where("user_id = ? AND scene = ? AND content_id = ?", userID, int32(interaction.Scene_ARTICLE), contentID).Take(&row).Error; err != nil {
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
	if err := db.Model(&model.ZfeedFavorite{}).Where("user_id = ? AND scene = ? AND content_id = ?", userID, int32(interaction.Scene_ARTICLE), contentID).Count(&count).Error; err != nil {
		t.Fatalf("count favorite rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("favorite row count = %d, want 0", count)
	}

	var eventCount int64
	if err := db.Model(&model.ZfeedFavoriteEvent{}).
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Count(&eventCount).Error; err != nil {
		t.Fatalf("count favorite event rows: %v", err)
	}
	if eventCount != 2 {
		t.Fatalf("favorite event row count = %d, want 2", eventCount)
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

func TestFavoriteReturnsNotFoundWhenTargetDoesNotExist(t *testing.T) {
	db := newFavoriteTestDB(t)
	_, client := newFavoriteTestRedis(t)

	logic := &FavoriteLogic{
		ctx:               context.Background(),
		svcCtx:            &svc.ServiceContext{MysqlDb: db, Redis: client},
		Logger:            logx.WithContext(context.Background()),
		favoriteRepo:      repositories.NewFavoriteRepository(context.Background(), db),
		favoriteEventRepo: repositories.NewFavoriteEventRepository(context.Background(), db),
		contentRepo: &favoriteStubContentRepo{
			getAuthorIDFunc: func(contentID int64) (int64, error) { return 0, nil },
		},
		commentRepo: &favoriteStubCommentRepo{},
	}

	_, err := logic.Favorite(&interaction.FavoriteReq{
		UserId:    1001,
		ContentId: 9001,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err == nil || !strings.Contains(err.Error(), "内容不存在") {
		t.Fatalf("Favorite error = %v, want contains 内容不存在", err)
	}
}

func TestFavoriteUsesResolvedOwnerInsteadOfClientValue(t *testing.T) {
	db := newFavoriteTestDB(t)
	_, client := newFavoriteTestRedis(t)

	logic := &FavoriteLogic{
		ctx:               context.Background(),
		svcCtx:            &svc.ServiceContext{MysqlDb: db, Redis: client},
		Logger:            logx.WithContext(context.Background()),
		favoriteRepo:      repositories.NewFavoriteRepository(context.Background(), db),
		favoriteEventRepo: repositories.NewFavoriteEventRepository(context.Background(), db),
		contentRepo: &favoriteStubContentRepo{
			getAuthorIDFunc: func(contentID int64) (int64, error) { return 2001, nil },
		},
		commentRepo: &favoriteStubCommentRepo{},
	}

	_, err := logic.Favorite(&interaction.FavoriteReq{
		UserId:    1001,
		ContentId: 9101,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("Favorite returned error: %v", err)
	}

	var row model.ZfeedFavorite
	if err := db.Where("user_id = ? AND scene = ? AND content_id = ?", 1001, int32(interaction.Scene_ARTICLE), 9101).Take(&row).Error; err != nil {
		t.Fatalf("query favorite row: %v", err)
	}
	if row.ContentUserID != 2001 {
		t.Fatalf("content_user_id = %d, want 2001", row.ContentUserID)
	}

	var eventRow model.ZfeedFavoriteEvent
	if err := db.Where("user_id = ? AND scene = ? AND content_id = ?", 1001, int32(interaction.Scene_ARTICLE), 9101).Take(&eventRow).Error; err != nil {
		t.Fatalf("query favorite event row: %v", err)
	}
	if eventRow.ContentUserID != 2001 {
		t.Fatalf("event content_user_id = %d, want 2001", eventRow.ContentUserID)
	}
}

func TestFavoriteQueriesSeparateSameContentIDByScene(t *testing.T) {
	db := newFavoriteTestDB(t)
	_, client := newFavoriteTestRedis(t)

	rows := []model.ZfeedFavorite{
		{UserID: 1001, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9401, ContentUserID: 2001, Status: repositories.FavoriteStatusActive},
		{UserID: 1002, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9401, ContentUserID: 2001, Status: repositories.FavoriteStatusActive},
		{UserID: 1001, Scene: int32(interaction.Scene_COMMENT), ContentID: 9401, ContentUserID: 3001, Status: repositories.FavoriteStatusActive},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed favorite rows: %v", err)
	}

	queryLogic := NewQueryFavoriteInfoLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	articleResp, err := queryLogic.QueryFavoriteInfo(&interaction.QueryFavoriteInfoReq{
		UserId:    1001,
		ContentId: 9401,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("query article favorite info: %v", err)
	}
	if articleResp.GetFavoriteCount() != 2 || !articleResp.GetIsFavorited() {
		t.Fatalf("article favorite info = %+v, want count=2 is_favorited=true", articleResp)
	}

	commentResp, err := queryLogic.QueryFavoriteInfo(&interaction.QueryFavoriteInfoReq{
		UserId:    1001,
		ContentId: 9401,
		Scene:     interaction.Scene_COMMENT,
	})
	if err != nil {
		t.Fatalf("query comment favorite info: %v", err)
	}
	if commentResp.GetFavoriteCount() != 1 || !commentResp.GetIsFavorited() {
		t.Fatalf("comment favorite info = %+v, want count=1 is_favorited=true", commentResp)
	}
}

func TestQueryFavoriteListOnlyReturnsContentScenes(t *testing.T) {
	db := newFavoriteTestDB(t)

	rows := []model.ZfeedFavorite{
		{ID: 10, UserID: 1001, Scene: int32(interaction.Scene_ARTICLE), ContentID: 9501, ContentUserID: 2001, Status: repositories.FavoriteStatusActive},
		{ID: 11, UserID: 1001, Scene: int32(interaction.Scene_COMMENT), ContentID: 9501, ContentUserID: 3001, Status: repositories.FavoriteStatusActive},
		{ID: 12, UserID: 1001, Scene: int32(interaction.Scene_VIDEO), ContentID: 9502, ContentUserID: 2002, Status: repositories.FavoriteStatusActive},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed favorite rows: %v", err)
	}

	logic := NewQueryFavoriteListLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	resp, err := logic.QueryFavoriteList(&interaction.QueryFavoriteListReq{
		UserId:   1001,
		Cursor:   0,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("QueryFavoriteList returned error: %v", err)
	}
	if len(resp.GetItems()) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.GetItems()))
	}
	if resp.GetItems()[0].GetContentId() != 9502 {
		t.Fatalf("first content_id = %d, want 9502", resp.GetItems()[0].GetContentId())
	}
	if resp.GetItems()[1].GetContentId() != 9501 {
		t.Fatalf("second content_id = %d, want 9501", resp.GetItems()[1].GetContentId())
	}
}

var _ repositories.ContentRepository = (*favoriteStubContentRepo)(nil)
var _ repositories.CommentRepository = (*favoriteStubCommentRepo)(nil)
