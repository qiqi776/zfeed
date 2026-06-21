package commentlogic

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"
)

func TestQueryCommentListCache(t *testing.T) {
	ctx := context.Background()
	_, redisClient := newCommentCacheRedis(t)
	logger := logx.WithContext(ctx)
	contentID := int64(9201)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	cachedItems := []*interaction.CommentItem{
		{CommentId: 303, ContentId: contentID, UserId: 403, Comment: "third"},
		{CommentId: 302, ContentId: contentID, UserId: 402, Comment: "second"},
		{CommentId: 301, ContentId: contentID, UserId: 401, Comment: "first"},
	}
	cmtCacheItemsAndIndex(ctx, logger, redisClient, cacheKey, cachedItems)

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{303, 302}) {
		t.Fatalf("cached comment ids = %v, want [303 302]", got)
	}
	if resp.GetNextCursor() != 302 || !resp.GetHasMore() {
		t.Fatalf("cached pagination = cursor:%d hasMore:%v, want 302 true", resp.GetNextCursor(), resp.GetHasMore())
	}
}

func TestQueryCommentListDB(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)
	contentID := int64(9301)

	if err := db.Create(&[]commentTestComment{
		{
			ID:         403,
			ContentID:  contentID,
			UserID:     503,
			Comment:    "third",
			Status:     repositories.CommentStatusNormal,
			ReplyCount: 1,
			CreatedAt:  time.Unix(1770000403, 0),
		},
		{
			ID:        402,
			ContentID: contentID,
			UserID:    502,
			Comment:   "second",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000402, 0),
		},
		{
			ID:        401,
			ContentID: contentID,
			UserID:    501,
			Comment:   "first",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000401, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed comments: %v", err)
	}

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				503: {UserId: 503, Nickname: "third-user", Avatar: "third.png"},
				502: {UserId: 502, Nickname: "second-user", Avatar: "second.png"},
			},
		},
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{403, 402}) {
		t.Fatalf("db comment ids = %v, want [403 402]", got)
	}
	if resp.GetNextCursor() != 402 || !resp.GetHasMore() {
		t.Fatalf("db pagination = cursor:%d hasMore:%v, want 402 true", resp.GetNextCursor(), resp.GetHasMore())
	}
	if resp.GetComments()[0].GetUserName() != "third-user" || resp.GetComments()[0].GetReplyCount() != 1 {
		t.Fatalf("first db item = %+v", resp.GetComments()[0])
	}
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	if !store.Exists(cacheKey) || !store.Exists(rediskey.BuildCommentItemKey("403")) {
		t.Fatalf("expected DB result to rebuild list and item cache")
	}
}

func TestQueryCommentListMiss(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	_, redisClient := newCommentCacheRedis(t)
	logger := logx.WithContext(ctx)
	contentID := int64(9351)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))

	cmtCacheItemsAndIndex(ctx, logger, redisClient, cacheKey, []*interaction.CommentItem{
		{CommentId: 703, ContentId: contentID, UserId: 803, Comment: "cached third", UserName: "cached-third"},
		{CommentId: 702, ContentId: contentID, UserId: 802, Comment: "stale middle"},
		{CommentId: 701, ContentId: contentID, UserId: 801, Comment: "cached first"},
	})
	if _, err := redisClient.DelCtx(ctx, rediskey.BuildCommentItemKey("702")); err != nil {
		t.Fatalf("delete middle item cache: %v", err)
	}

	if err := db.Create(&commentTestComment{
		ID:         702,
		ContentID:  contentID,
		UserID:     802,
		Comment:    "refilled middle",
		Status:     repositories.CommentStatusNormal,
		ReplyCount: 4,
		CreatedAt:  time.Unix(1770000702, 0),
	}).Error; err != nil {
		t.Fatalf("seed missing comment: %v", err)
	}

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				802: {UserId: 802, Nickname: "middle-user", Avatar: "middle.png"},
			},
		},
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{703, 702}) {
		t.Fatalf("refilled cached comment ids = %v, want [703 702]", got)
	}
	if resp.GetNextCursor() != 702 || !resp.GetHasMore() {
		t.Fatalf("refilled pagination = cursor:%d hasMore:%v, want 702 true", resp.GetNextCursor(), resp.GetHasMore())
	}
	refilled := resp.GetComments()[1]
	if refilled.GetComment() != "refilled middle" ||
		refilled.GetUserName() != "middle-user" ||
		refilled.GetUserAvatar() != "middle.png" ||
		refilled.GetReplyCount() != 4 {
		t.Fatalf("refilled item = %+v", refilled)
	}
	if exists, err := redisClient.ExistsCtx(ctx, rediskey.BuildCommentItemKey("702")); err != nil || !exists {
		t.Fatalf("expected refilled item cache to exist, exists:%v err:%v", exists, err)
	}
}

func TestQueryCommentListStale(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	_, redisClient := newCommentCacheRedis(t)
	logger := logx.WithContext(ctx)
	contentID := int64(9361)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))

	cmtCacheItemsAndIndex(ctx, logger, redisClient, cacheKey, []*interaction.CommentItem{
		{CommentId: 713, ContentId: contentID + 1, UserId: 813, Comment: "stale other content"},
	})

	if err := db.Create(&commentTestComment{
		ID:        712,
		ContentID: contentID,
		UserID:    812,
		Comment:   "fresh current content",
		Status:    repositories.CommentStatusNormal,
		CreatedAt: time.Unix(1770000712, 0),
	}).Error; err != nil {
		t.Fatalf("seed current comment: %v", err)
	}

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				812: {UserId: 812, Nickname: "current-user", Avatar: "current.png"},
			},
		},
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{712}) {
		t.Fatalf("comment ids after stale cache = %v, want [712]", got)
	}
	if resp.GetComments()[0].GetComment() != "fresh current content" || resp.GetComments()[0].GetUserName() != "current-user" {
		t.Fatalf("fallback comment = %+v", resp.GetComments()[0])
	}
}

func TestQueryCommentListMissing(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)
	contentID := int64(9365)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	lockKey := rediskey.BuildCommentListLockKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	store.ZAdd(cacheKey, 99999, "99999")
	store.Set(lockKey, "busy")

	if err := db.Create(&commentTestComment{
		ID:        716,
		ContentID: contentID,
		UserID:    816,
		Comment:   "fallback current content",
		Status:    repositories.CommentStatusNormal,
		CreatedAt: time.Unix(1770000716, 0),
	}).Error; err != nil {
		t.Fatalf("seed current comment: %v", err)
	}

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				816: {UserId: 816, Nickname: "fallback-user", Avatar: "fallback.png"},
			},
		},
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{716}) {
		t.Fatalf("comment ids after missing cache = %v, want [716]", got)
	}
	if resp.GetComments()[0].GetUserName() != "fallback-user" {
		t.Fatalf("fallback user = %+v", resp.GetComments()[0])
	}
	if store.Exists(cacheKey) {
		t.Fatal("expected broken comment list cache to be invalidated")
	}
}

func TestQueryCommentListRefillError(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	_, redisClient := newCommentCacheRedis(t)
	contentID := int64(9367)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	if _, err := redisClient.ZaddCtx(ctx, cacheKey, 936701, "936701"); err != nil {
		t.Fatalf("seed stale index: %v", err)
	}

	if err := db.Create(&[]commentTestComment{
		{
			ID:        936701,
			ContentID: contentID + 1,
			UserID:    836701,
			Comment:   "stale refill source",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770006701, 0),
		},
		{
			ID:        936702,
			ContentID: contentID,
			UserID:    836702,
			Comment:   "fallback deleted comment",
			Status:    repositories.CommentStatusDeleted,
			CreatedAt: time.Unix(1770006702, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed fallback comment: %v", err)
	}
	userRPC := &fakeCommentUserService{
		users: map[int64]*userservice.UserInfo{
			836701: {UserId: 836701, Nickname: "stale-user"},
		},
		failures: 1,
		failErr:  errors.New("refill user rpc unavailable"),
	}

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: userRPC,
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{936702}) {
		t.Fatalf("comment ids after refill error = %v, want [936702]", got)
	}
	item := resp.GetComments()[0]
	if item.GetContentId() != contentID ||
		item.GetComment() != "该评论已删除" ||
		item.GetStatus() != repositories.CommentStatusDeleted ||
		item.GetUserId() != 0 {
		t.Fatalf("fallback comment item = %+v", item)
	}
	if userRPC.failures != 0 {
		t.Fatal("expected refill user failure to be consumed")
	}
}

func TestQueryCommentListLockError(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	redisClient := newErrorCommentRedis(t)
	contentID := int64(9369)

	if err := db.Create(&[]commentTestComment{
		{
			ID:        936902,
			ContentID: contentID,
			UserID:    836902,
			Comment:   "lock fallback second",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770006902, 0),
		},
		{
			ID:        936901,
			ContentID: contentID,
			UserID:    836901,
			Comment:   "lock fallback first",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770006901, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed lock fallback comments: %v", err)
	}

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				836902: {UserId: 836902, Nickname: "lock-second"},
				836901: {UserId: 836901, Nickname: "lock-first"},
			},
		},
	}).queryWithRebuild(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  1,
	}, 1)
	if err != nil {
		t.Fatalf("queryWithRebuild returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{936902}) {
		t.Fatalf("lock fallback comment ids = %v, want [936902]", got)
	}
	if resp.GetNextCursor() != 936902 || !resp.GetHasMore() {
		t.Fatalf("lock fallback pagination = cursor:%d hasMore:%v", resp.GetNextCursor(), resp.GetHasMore())
	}
	if resp.GetComments()[0].GetContentId() != contentID || resp.GetComments()[0].GetUserName() != "lock-second" {
		t.Fatalf("lock fallback comment item = %+v", resp.GetComments()[0])
	}
}

func TestQueryCommentListCacheDBResult(t *testing.T) {
	ctx := context.Background()
	store, redisClient := newCommentCacheRedis(t)
	contentID := int64(9370)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	logic := NewQueryCommentListLogic(ctx, &svc.ServiceContext{Redis: redisClient})

	logic.cacheDBResult(interaction.Scene_ARTICLE, contentID, nil)
	if store.Exists(cacheKey) {
		t.Fatal("empty comment DB result should not create list cache")
	}

	logic.cacheDBResult(interaction.Scene_ARTICLE, contentID, []*interaction.CommentItem{
		{CommentId: 937001, ContentId: contentID, UserId: 837001, Comment: "cached db result"},
	})
	if !store.Exists(cacheKey) || !store.Exists(rediskey.BuildCommentItemKey("937001")) {
		t.Fatalf("expected comment DB result cache to exist: list=%v item=%v", store.Exists(cacheKey), store.Exists(rediskey.BuildCommentItemKey("937001")))
	}
}

func TestQueryCommentListWaitCache(t *testing.T) {
	ctx := context.Background()
	store, redisClient := newCommentCacheRedis(t)
	contentID := int64(9731)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	lockKey := rediskey.BuildCommentListLockKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	if err := redisClient.SetexCtx(ctx, lockKey, "locked", rediskey.CommentLockExpireSecs); err != nil {
		t.Fatalf("seed rebuild lock: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		payload, _ := marshalCommentItem(&interaction.CommentItem{
			CommentId: 973101,
			ContentId: contentID,
			UserId:    973201,
			Comment:   "cached after wait",
		})
		_ = store.Set(rediskey.BuildCommentItemKey("973101"), payload)
		store.ZAdd(cacheKey, float64(973101), "973101")
	}()

	repo := &fakeCommentRepository{listRootErr: errors.New("unexpected db fallback")}
	logic := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	})
	logic.commentRepo = repo

	resp, err := logic.QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  1,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{973101}) {
		t.Fatalf("wait cache comment ids = %v, want [973101]", got)
	}
	if resp.GetComments()[0].GetComment() != "cached after wait" {
		t.Fatalf("wait cache comment = %+v", resp.GetComments()[0])
	}
	if repo.listRootCalls != 0 {
		t.Fatalf("ListRootCommentsIncludeDeleted calls = %d, want 0", repo.listRootCalls)
	}
}

func TestQueryCommentListDBError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("root comments unavailable")
	logic := NewQueryCommentListLogic(ctx, &svc.ServiceContext{})
	logic.commentRepo = &fakeCommentRepository{listRootErr: repoErr}

	resp, err := logic.queryDB(&interaction.QueryCommentListReq{
		ContentId: 9371,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	}, 2, false)
	if err == nil {
		t.Fatal("expected queryDB error")
	}
	if resp != nil {
		t.Fatalf("queryDB response = %+v, want nil", resp)
	}
}

func TestQueryCommentListUserError(t *testing.T) {
	ctx := context.Background()
	userErr := errors.New("user rpc unavailable")
	logic := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		UserRpc: &fakeCommentUserService{err: userErr},
	})
	logic.commentRepo = &fakeCommentRepository{
		listRoot: []*do.CommentDO{
			{
				ID:        722,
				ContentID: 9372,
				UserID:    822,
				Comment:   "needs user",
				Status:    repositories.CommentStatusNormal,
				CreatedAt: time.Unix(1770000722, 0),
			},
		},
	}

	resp, err := logic.queryDB(&interaction.QueryCommentListReq{
		ContentId: 9372,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	}, 2, false)
	if err == nil {
		t.Fatal("expected queryDB user error")
	}
	if resp != nil {
		t.Fatalf("queryDB response = %+v, want nil", resp)
	}
}

func TestQueryCommentListUserBizError(t *testing.T) {
	ctx := context.Background()
	userErr := errorx.NewBadRequest("用户参数错误")
	logic := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		UserRpc: &fakeCommentUserService{err: userErr},
	})
	logic.commentRepo = &fakeCommentRepository{
		listRoot: []*do.CommentDO{
			{
				ID:        937301,
				ContentID: 9373,
				UserID:    837301,
				Comment:   "needs user biz error",
				Status:    repositories.CommentStatusNormal,
				CreatedAt: time.Unix(1770007301, 0),
			},
		},
	}

	resp, err := logic.queryDB(&interaction.QueryCommentListReq{
		ContentId: 9373,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	}, 2, false)
	if !errors.Is(err, userErr) {
		t.Fatalf("queryDB error = %v, want %v", err, userErr)
	}
	if resp != nil {
		t.Fatalf("queryDB response = %+v, want nil", resp)
	}
}

func TestQueryCommentListRejects(t *testing.T) {
	ctx := context.Background()
	store, redisClient := newCommentCacheRedis(t)
	contentID := int64(9401)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	store.ZAdd(cacheKey, 100, "100")

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		Cursor:    50,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if len(resp.GetComments()) != 0 || resp.GetNextCursor() != 0 || resp.GetHasMore() {
		t.Fatalf("empty cache response = %+v, want empty page", resp)
	}

	logic := NewQueryCommentListLogic(ctx, &svc.ServiceContext{})
	for _, req := range []*interaction.QueryCommentListReq{
		nil,
		{ContentId: 0, Scene: interaction.Scene_ARTICLE, PageSize: 10},
		{ContentId: 1, Scene: interaction.Scene_SCENE_UNKNOWN, PageSize: 10},
		{ContentId: 1, Scene: interaction.Scene_ARTICLE, PageSize: 0},
	} {
		if _, err := logic.QueryCommentList(req); err == nil {
			t.Fatalf("expected QueryCommentList error for request %+v", req)
		}
	}
}

func TestQueryCommentListBadIndex(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)
	contentID := int64(9381)
	cacheKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	store.ZAdd(cacheKey, 938101, "not-a-comment-id")

	if err := db.Create(&commentTestComment{
		ID:        938101,
		ContentID: contentID,
		UserID:    838101,
		Comment:   "db comment hidden by bad index",
		Status:    repositories.CommentStatusNormal,
		CreatedAt: time.Unix(1770008101, 0),
	}).Error; err != nil {
		t.Fatalf("seed fallback comment: %v", err)
	}

	resp, err := NewQueryCommentListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				838101: {UserId: 838101, Nickname: "bad-index-user"},
			},
		},
	}).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		PageSize:  2,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{938101}) {
		t.Fatalf("comment ids after bad index = %v, want [938101]", got)
	}
	if resp.GetComments()[0].GetUserName() != "bad-index-user" {
		t.Fatalf("fallback user = %+v", resp.GetComments()[0])
	}
	members, err := store.ZMembers(cacheKey)
	if err != nil {
		t.Fatalf("read rebuilt comment list index: %v", err)
	}
	if !reflect.DeepEqual(members, []string{"938101"}) {
		t.Fatalf("rebuilt comment list index = %v, want [938101]", members)
	}
}
