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

func TestQueryReplyListCacheAndDB(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	_, redisClient := newCommentCacheRedis(t)
	logger := logx.WithContext(ctx)
	rootID := int64(9501)
	cachedKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	cachedItems := []*interaction.CommentItem{
		{CommentId: 503, RootId: rootID, UserId: 603, Comment: "cached third"},
		{CommentId: 502, RootId: rootID, UserId: 602, Comment: "cached second"},
	}
	cmtCacheItemsAndIndex(ctx, logger, redisClient, cachedKey, cachedItems)

	cachedResp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("cached QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(cachedResp.GetReplies()); !reflect.DeepEqual(got, []int64{503}) {
		t.Fatalf("cached reply ids = %v, want [503]", got)
	}
	if cachedResp.GetRootId() != rootID || cachedResp.GetNextCursor() != 503 || !cachedResp.GetHasMore() {
		t.Fatalf("cached reply pagination = %+v", cachedResp)
	}

	dbRootID := int64(9601)
	if err := db.Create(&[]commentTestComment{
		{
			ID:            604,
			ContentID:     9701,
			UserID:        704,
			ReplyToUserID: 801,
			ParentID:      dbRootID,
			RootID:        dbRootID,
			Comment:       "db fourth",
			Status:        repositories.CommentStatusNormal,
			CreatedAt:     time.Unix(1770000604, 0),
		},
		{
			ID:        603,
			ContentID: 9701,
			UserID:    703,
			ParentID:  dbRootID,
			RootID:    dbRootID,
			Comment:   "db third",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000603, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed replies: %v", err)
	}

	dbResp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				704: {UserId: 704, Nickname: "reply-four"},
			},
		},
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   dbRootID,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("db QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(dbResp.GetReplies()); !reflect.DeepEqual(got, []int64{604, 603}) {
		t.Fatalf("db reply ids = %v, want [604 603]", got)
	}
	if dbResp.GetRootId() != dbRootID || dbResp.GetHasMore() {
		t.Fatalf("db reply response = %+v", dbResp)
	}
	if dbResp.GetReplies()[0].GetUserName() != "reply-four" || dbResp.GetReplies()[0].GetReplyToUserId() != 801 {
		t.Fatalf("db reply item = %+v", dbResp.GetReplies()[0])
	}
}

func TestQueryReplyListMiss(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	_, redisClient := newCommentCacheRedis(t)
	logger := logx.WithContext(ctx)
	rootID := int64(9651)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))

	cmtCacheItemsAndIndex(ctx, logger, redisClient, cacheKey, []*interaction.CommentItem{
		{CommentId: 803, RootId: rootID, UserId: 903, Comment: "cached third"},
		{CommentId: 802, RootId: rootID, UserId: 902, Comment: "stale middle"},
		{CommentId: 801, RootId: rootID, UserId: 901, Comment: "cached first"},
	})
	if _, err := redisClient.DelCtx(ctx, rediskey.BuildCommentItemKey("802")); err != nil {
		t.Fatalf("delete middle reply cache: %v", err)
	}

	if err := db.Create(&commentTestComment{
		ID:            802,
		ContentID:     9751,
		UserID:        902,
		ReplyToUserID: 9902,
		ParentID:      rootID,
		RootID:        rootID,
		Comment:       "refilled reply",
		Status:        repositories.CommentStatusNormal,
		CreatedAt:     time.Unix(1770000802, 0),
	}).Error; err != nil {
		t.Fatalf("seed missing reply: %v", err)
	}

	resp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				902: {UserId: 902, Nickname: "reply-user", Avatar: "reply.png"},
			},
		},
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetReplies()); !reflect.DeepEqual(got, []int64{803, 802}) {
		t.Fatalf("refilled reply ids = %v, want [803 802]", got)
	}
	if resp.GetRootId() != rootID || resp.GetNextCursor() != 802 || !resp.GetHasMore() {
		t.Fatalf("refilled reply pagination = %+v", resp)
	}
	refilled := resp.GetReplies()[1]
	if refilled.GetComment() != "refilled reply" ||
		refilled.GetUserName() != "reply-user" ||
		refilled.GetUserAvatar() != "reply.png" ||
		refilled.GetReplyToUserId() != 9902 ||
		refilled.GetParentId() != rootID ||
		refilled.GetRootId() != rootID {
		t.Fatalf("refilled reply = %+v", refilled)
	}
	if exists, err := redisClient.ExistsCtx(ctx, rediskey.BuildCommentItemKey("802")); err != nil || !exists {
		t.Fatalf("expected refilled reply cache to exist, exists:%v err:%v", exists, err)
	}
}

func TestQueryReplyListStale(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	_, redisClient := newCommentCacheRedis(t)
	logger := logx.WithContext(ctx)
	rootID := int64(9661)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))

	cmtCacheItemsAndIndex(ctx, logger, redisClient, cacheKey, []*interaction.CommentItem{
		{CommentId: 813, RootId: rootID + 1, UserId: 913, Comment: "stale other root"},
	})

	if err := db.Create(&commentTestComment{
		ID:        812,
		ContentID: 9761,
		UserID:    912,
		ParentID:  rootID,
		RootID:    rootID,
		Comment:   "fresh current reply",
		Status:    repositories.CommentStatusNormal,
		CreatedAt: time.Unix(1770000812, 0),
	}).Error; err != nil {
		t.Fatalf("seed current reply: %v", err)
	}

	resp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				912: {UserId: 912, Nickname: "current-reply-user", Avatar: "current-reply.png"},
			},
		},
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetReplies()); !reflect.DeepEqual(got, []int64{812}) {
		t.Fatalf("reply ids after stale cache = %v, want [812]", got)
	}
	if resp.GetRootId() != rootID ||
		resp.GetReplies()[0].GetComment() != "fresh current reply" ||
		resp.GetReplies()[0].GetUserName() != "current-reply-user" {
		t.Fatalf("fallback reply = %+v", resp)
	}
}

func TestQueryReplyListMissing(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)
	rootID := int64(9665)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	lockKey := rediskey.BuildCommentReplyLockKey(strconv.FormatInt(rootID, 10))
	store.ZAdd(cacheKey, 99998, "99998")
	store.Set(lockKey, "busy")

	if err := db.Create(&commentTestComment{
		ID:        816,
		ContentID: 9765,
		UserID:    916,
		ParentID:  rootID,
		RootID:    rootID,
		Comment:   "fallback current reply",
		Status:    repositories.CommentStatusNormal,
		CreatedAt: time.Unix(1770000816, 0),
	}).Error; err != nil {
		t.Fatalf("seed current reply: %v", err)
	}

	resp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				916: {UserId: 916, Nickname: "fallback-reply-user", Avatar: "fallback-reply.png"},
			},
		},
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetReplies()); !reflect.DeepEqual(got, []int64{816}) {
		t.Fatalf("reply ids after missing cache = %v, want [816]", got)
	}
	if resp.GetRootId() != rootID || resp.GetReplies()[0].GetUserName() != "fallback-reply-user" {
		t.Fatalf("fallback reply = %+v", resp)
	}
	if store.Exists(cacheKey) {
		t.Fatal("expected broken reply list cache to be invalidated")
	}
}

func TestQueryReplyListRefillError(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	_, redisClient := newCommentCacheRedis(t)
	rootID := int64(9667)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	if _, err := redisClient.ZaddCtx(ctx, cacheKey, 966701, "966701"); err != nil {
		t.Fatalf("seed stale reply index: %v", err)
	}

	if err := db.Create(&[]commentTestComment{
		{
			ID:        966701,
			ContentID: 9767,
			UserID:    866701,
			ParentID:  rootID + 1,
			RootID:    rootID + 1,
			Comment:   "stale refill reply",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770006702, 0),
		},
		{
			ID:        966702,
			ContentID: 9767,
			UserID:    866702,
			ParentID:  rootID,
			RootID:    rootID,
			Comment:   "fallback deleted reply",
			Status:    repositories.CommentStatusDeleted,
			CreatedAt: time.Unix(1770006703, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed fallback reply: %v", err)
	}
	userRPC := &fakeCommentUserService{
		users: map[int64]*userservice.UserInfo{
			866701: {UserId: 866701, Nickname: "stale-reply-user"},
		},
		failures: 1,
		failErr:  errors.New("refill reply user rpc unavailable"),
	}

	resp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: userRPC,
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetReplies()); !reflect.DeepEqual(got, []int64{966702}) {
		t.Fatalf("reply ids after refill error = %v, want [966702]", got)
	}
	item := resp.GetReplies()[0]
	if resp.GetRootId() != rootID ||
		item.GetRootId() != rootID ||
		item.GetComment() != "该评论已删除" ||
		item.GetStatus() != repositories.CommentStatusDeleted ||
		item.GetUserId() != 0 {
		t.Fatalf("fallback reply item = %+v", item)
	}
	if userRPC.failures != 0 {
		t.Fatal("expected refill reply user failure to be consumed")
	}
}

func TestQueryReplyListLockError(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	redisClient := newErrorCommentRedis(t)
	rootID := int64(9669)

	if err := db.Create(&[]commentTestComment{
		{
			ID:        966902,
			ContentID: 9769,
			UserID:    866902,
			ParentID:  rootID,
			RootID:    rootID,
			Comment:   "lock fallback reply second",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770006902, 0),
		},
		{
			ID:        966901,
			ContentID: 9769,
			UserID:    866901,
			ParentID:  rootID,
			RootID:    rootID,
			Comment:   "lock fallback reply first",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770006901, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed lock fallback replies: %v", err)
	}

	resp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				866902: {UserId: 866902, Nickname: "lock-reply-second"},
				866901: {UserId: 866901, Nickname: "lock-reply-first"},
			},
		},
	}).queryWithRebuild(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 1,
	}, 1)
	if err != nil {
		t.Fatalf("queryWithRebuild returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetReplies()); !reflect.DeepEqual(got, []int64{966902}) {
		t.Fatalf("lock fallback reply ids = %v, want [966902]", got)
	}
	if resp.GetRootId() != rootID || resp.GetNextCursor() != 966902 || !resp.GetHasMore() {
		t.Fatalf("lock fallback reply pagination = %+v", resp)
	}
	if resp.GetReplies()[0].GetRootId() != rootID || resp.GetReplies()[0].GetUserName() != "lock-reply-second" {
		t.Fatalf("lock fallback reply item = %+v", resp.GetReplies()[0])
	}
}

func TestQueryReplyListCacheDBResult(t *testing.T) {
	ctx := context.Background()
	store, redisClient := newCommentCacheRedis(t)
	rootID := int64(9670)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	logic := NewQueryReplyListLogic(ctx, &svc.ServiceContext{Redis: redisClient})

	logic.cacheDBResult(rootID, nil)
	if store.Exists(cacheKey) {
		t.Fatal("empty reply DB result should not create reply cache")
	}

	logic.cacheDBResult(rootID, []*interaction.CommentItem{
		{CommentId: 967001, RootId: rootID, UserId: 867001, Comment: "cached reply result"},
	})
	if !store.Exists(cacheKey) || !store.Exists(rediskey.BuildCommentItemKey("967001")) {
		t.Fatalf("expected reply DB result cache to exist: list=%v item=%v", store.Exists(cacheKey), store.Exists(rediskey.BuildCommentItemKey("967001")))
	}
}

func TestQueryReplyListWaitCache(t *testing.T) {
	ctx := context.Background()
	store, redisClient := newCommentCacheRedis(t)
	rootID := int64(9741)
	replyID := int64(974101)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	lockKey := rediskey.BuildCommentReplyLockKey(strconv.FormatInt(rootID, 10))
	if err := redisClient.SetexCtx(ctx, lockKey, "locked", rediskey.CommentLockExpireSecs); err != nil {
		t.Fatalf("seed rebuild lock: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		payload, _ := marshalCommentItem(&interaction.CommentItem{
			CommentId: replyID,
			RootId:    rootID,
			UserId:    974201,
			Comment:   "reply cached after wait",
		})
		_ = store.Set(rediskey.BuildCommentItemKey(strconv.FormatInt(replyID, 10)), payload)
		store.ZAdd(cacheKey, float64(replyID), strconv.FormatInt(replyID, 10))
	}()

	repo := &fakeCommentRepository{listRepliesErr: errors.New("unexpected db fallback")}
	logic := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	})
	logic.commentRepo = repo

	resp, err := logic.QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetReplies()); !reflect.DeepEqual(got, []int64{replyID}) {
		t.Fatalf("wait cache reply ids = %v, want [%d]", got, replyID)
	}
	if resp.GetReplies()[0].GetComment() != "reply cached after wait" {
		t.Fatalf("wait cache reply = %+v", resp.GetReplies()[0])
	}
	if repo.listRepliesCalls != 0 {
		t.Fatalf("ListRepliesIncludeDeleted calls = %d, want 0", repo.listRepliesCalls)
	}
}

func TestQueryReplyListDBError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("replies unavailable")
	logic := NewQueryReplyListLogic(ctx, &svc.ServiceContext{})
	logic.commentRepo = &fakeCommentRepository{listRepliesErr: repoErr}

	resp, err := logic.queryDB(&interaction.QueryReplyListReq{
		RootId:   9671,
		PageSize: 2,
	}, 2, false)
	if err == nil {
		t.Fatal("expected queryDB error")
	}
	if resp != nil {
		t.Fatalf("queryDB response = %+v, want nil", resp)
	}
}

func TestQueryReplyListUserError(t *testing.T) {
	ctx := context.Background()
	userErr := errors.New("user rpc unavailable")
	logic := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		UserRpc: &fakeCommentUserService{err: userErr},
	})
	logic.commentRepo = &fakeCommentRepository{
		listReplies: []*do.CommentDO{
			{
				ID:        822,
				ContentID: 9772,
				UserID:    922,
				ParentID:  9672,
				RootID:    9672,
				Comment:   "reply needs user",
				Status:    repositories.CommentStatusNormal,
				CreatedAt: time.Unix(1770000822, 0),
			},
		},
	}

	resp, err := logic.queryDB(&interaction.QueryReplyListReq{
		RootId:   9672,
		PageSize: 2,
	}, 2, false)
	if err == nil {
		t.Fatal("expected queryDB user error")
	}
	if resp != nil {
		t.Fatalf("queryDB response = %+v, want nil", resp)
	}
}

func TestQueryReplyListUserBizError(t *testing.T) {
	ctx := context.Background()
	userErr := errorx.NewBadRequest("用户参数错误")
	logic := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		UserRpc: &fakeCommentUserService{err: userErr},
	})
	logic.commentRepo = &fakeCommentRepository{
		listReplies: []*do.CommentDO{
			{
				ID:        967301,
				ContentID: 9773,
				UserID:    867301,
				ParentID:  9673,
				RootID:    9673,
				Comment:   "reply needs user biz error",
				Status:    repositories.CommentStatusNormal,
				CreatedAt: time.Unix(1770007302, 0),
			},
		},
	}

	resp, err := logic.queryDB(&interaction.QueryReplyListReq{
		RootId:   9673,
		PageSize: 2,
	}, 2, false)
	if !errors.Is(err, userErr) {
		t.Fatalf("queryDB error = %v, want %v", err, userErr)
	}
	if resp != nil {
		t.Fatalf("queryDB response = %+v, want nil", resp)
	}
}

func TestQueryReplyListRejects(t *testing.T) {
	ctx := context.Background()
	store, redisClient := newCommentCacheRedis(t)
	rootID := int64(9701)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	store.ZAdd(cacheKey, 100, "100")

	resp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		Cursor:   50,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("QueryReplyList returned error: %v", err)
	}
	if resp.GetRootId() != rootID || len(resp.GetReplies()) != 0 || resp.GetNextCursor() != 0 || resp.GetHasMore() {
		t.Fatalf("empty reply cache response = %+v, want empty page", resp)
	}

	logic := NewQueryReplyListLogic(ctx, &svc.ServiceContext{})
	for _, req := range []*interaction.QueryReplyListReq{
		nil,
		{RootId: 0, PageSize: 10},
		{RootId: 1, PageSize: 0},
	} {
		if _, err := logic.QueryReplyList(req); err == nil {
			t.Fatalf("expected QueryReplyList error for request %+v", req)
		}
	}
}

func TestQueryReplyListBadIndex(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)
	rootID := int64(9681)
	replyID := int64(968101)
	cacheKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	store.ZAdd(cacheKey, float64(replyID), "not-a-comment-id")

	if err := db.Create(&commentTestComment{
		ID:        replyID,
		ContentID: 9781,
		UserID:    868101,
		ParentID:  rootID,
		RootID:    rootID,
		Comment:   "db reply hidden by bad index",
		Status:    repositories.CommentStatusNormal,
		CreatedAt: time.Unix(1770008102, 0),
	}).Error; err != nil {
		t.Fatalf("seed fallback reply: %v", err)
	}

	resp, err := NewQueryReplyListLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				868101: {UserId: 868101, Nickname: "bad-index-reply-user"},
			},
		},
	}).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootID,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("QueryReplyList returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetReplies()); !reflect.DeepEqual(got, []int64{replyID}) {
		t.Fatalf("reply ids after bad index = %v, want [%d]", got, replyID)
	}
	if resp.GetRootId() != rootID || resp.GetReplies()[0].GetUserName() != "bad-index-reply-user" {
		t.Fatalf("fallback reply = %+v", resp)
	}
	members, err := store.ZMembers(cacheKey)
	if err != nil {
		t.Fatalf("read rebuilt reply list index: %v", err)
	}
	if !reflect.DeepEqual(members, []string{"968101"}) {
		t.Fatalf("rebuilt reply list index = %v, want [968101]", members)
	}
}
