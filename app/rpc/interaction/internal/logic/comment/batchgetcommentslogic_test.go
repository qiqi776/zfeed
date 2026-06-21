package commentlogic

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/app/rpc/user/client/userservice"
)

func TestBatchGetCommentsCacheAndDB(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)

	if err := db.Create(&[]commentTestComment{
		{
			ID:        101,
			ContentID: 9001,
			UserID:    201,
			Comment:   "db comment",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000200, 0),
		},
		{
			ID:        102,
			ContentID: 9001,
			UserID:    202,
			Comment:   "deleted db comment",
			Status:    repositories.CommentStatusDeleted,
			IsDeleted: 1,
			CreatedAt: time.Unix(1770000201, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed comments: %v", err)
	}

	cachedPayload, err := marshalCommentItem(&interaction.CommentItem{
		CommentId: 100,
		ContentId: 9001,
		UserId:    200,
		Comment:   "cached comment",
	})
	if err != nil {
		t.Fatalf("marshal cached item: %v", err)
	}
	if err := redisClient.SetCtx(ctx, rediskey.BuildCommentItemKey("100"), cachedPayload); err != nil {
		t.Fatalf("seed cached item: %v", err)
	}

	logic := NewBatchGetCommentsLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				201: {UserId: 201, Nickname: "db-user", Avatar: "db.png"},
			},
		},
	})

	resp, err := logic.BatchGetComments(&interaction.BatchGetCommentsReq{
		CommentIds: []int64{0, 100, 101, 102, 999, 101},
	})
	if err != nil {
		t.Fatalf("BatchGetComments returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{100, 101, 102}) {
		t.Fatalf("comment ids = %v, want [100 101 102]", got)
	}
	if !reflect.DeepEqual(resp.GetMissIds(), []int64{999}) {
		t.Fatalf("miss ids = %v, want [999]", resp.GetMissIds())
	}
	if resp.GetComments()[1].GetUserName() != "db-user" {
		t.Fatalf("db comment username = %q, want db-user", resp.GetComments()[1].GetUserName())
	}
	if resp.GetComments()[2].GetUserId() != 0 || resp.GetComments()[2].GetComment() != "该评论已删除" {
		t.Fatalf("deleted db item = %+v", resp.GetComments()[2])
	}

	if !store.Exists(rediskey.BuildCommentItemKey("101")) {
		t.Fatal("expected DB comment to be written back to cache")
	}
}

func TestBatchGetCommentsEmpty(t *testing.T) {
	ctx := context.Background()
	logic := NewBatchGetCommentsLogic(ctx, &svc.ServiceContext{})

	if _, err := logic.BatchGetComments(nil); err == nil {
		t.Fatal("expected nil request error")
	}

	resp, err := logic.BatchGetComments(&interaction.BatchGetCommentsReq{
		CommentIds: []int64{0, -1},
	})
	if err != nil {
		t.Fatalf("empty ids returned error: %v", err)
	}
	if len(resp.GetComments()) != 0 || len(resp.GetMissIds()) != 0 {
		t.Fatalf("empty ids response = %+v, want empty comments and misses", resp)
	}
}

func TestBatchGetCommentsDBError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("batch repository unavailable")
	logic := NewBatchGetCommentsLogic(ctx, &svc.ServiceContext{})
	logic.commentRepo = &fakeCommentRepository{batchErr: repoErr}

	resp, err := logic.BatchGetComments(&interaction.BatchGetCommentsReq{
		CommentIds: []int64{301},
	})
	if err == nil {
		t.Fatal("expected BatchGetComments error")
	}
	if resp != nil {
		t.Fatalf("BatchGetComments response = %+v, want nil", resp)
	}
}

func TestBatchGetCommentsCacheError(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	redisClient := newErrorCommentRedis(t)

	if err := db.Create(&[]commentTestComment{
		{
			ID:        501,
			ContentID: 9501,
			UserID:    601,
			Comment:   "db first after cache error",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000501, 0),
		},
		{
			ID:        502,
			ContentID: 9501,
			UserID:    602,
			Comment:   "db second after cache error",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000502, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed comments: %v", err)
	}

	resp, err := NewBatchGetCommentsLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				601: {UserId: 601, Nickname: "cache-error-first"},
				602: {UserId: 602, Nickname: "cache-error-second"},
			},
		},
	}).BatchGetComments(&interaction.BatchGetCommentsReq{
		CommentIds: []int64{502, 999, 501, 502},
	})
	if err != nil {
		t.Fatalf("BatchGetComments returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{502, 501}) {
		t.Fatalf("comment ids after cache error = %v, want [502 501]", got)
	}
	if !reflect.DeepEqual(resp.GetMissIds(), []int64{999}) {
		t.Fatalf("miss ids after cache error = %v, want [999]", resp.GetMissIds())
	}
	if resp.GetComments()[0].GetUserName() != "cache-error-second" ||
		resp.GetComments()[1].GetUserName() != "cache-error-first" {
		t.Fatalf("comment users after cache error = [%q %q]", resp.GetComments()[0].GetUserName(), resp.GetComments()[1].GetUserName())
	}
}
