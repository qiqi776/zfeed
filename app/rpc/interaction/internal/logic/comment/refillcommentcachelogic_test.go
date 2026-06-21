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

func TestRefillCommentCacheOrders(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)

	if err := db.Create(&[]commentTestComment{
		{
			ID:        201,
			ContentID: 9101,
			UserID:    301,
			Comment:   "first",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000300, 0),
		},
		{
			ID:        202,
			ContentID: 9101,
			UserID:    302,
			Comment:   "second",
			Status:    repositories.CommentStatusNormal,
			CreatedAt: time.Unix(1770000301, 0),
		},
	}).Error; err != nil {
		t.Fatalf("seed comments: %v", err)
	}

	logic := NewRefillCommentCacheLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				301: {UserId: 301, Nickname: "first-user"},
				302: {UserId: 302, Nickname: "second-user"},
			},
		},
	})

	resp, err := logic.RefillCommentCache(&interaction.RefillCommentCacheReq{
		CommentIds: []int64{202, 999, 201, 202},
	})
	if err != nil {
		t.Fatalf("RefillCommentCache returned error: %v", err)
	}
	if got := commentItemIDs(resp.GetComments()); !reflect.DeepEqual(got, []int64{202, 201}) {
		t.Fatalf("refill ids = %v, want [202 201]", got)
	}
	if resp.GetComments()[0].GetUserName() != "second-user" || resp.GetComments()[1].GetUserName() != "first-user" {
		t.Fatalf("refill usernames = [%q %q]", resp.GetComments()[0].GetUserName(), resp.GetComments()[1].GetUserName())
	}
	if !store.Exists(rediskey.BuildCommentItemKey("201")) || !store.Exists(rediskey.BuildCommentItemKey("202")) {
		t.Fatal("expected refill comments to be written to cache")
	}
}

func TestRefillCommentCacheEmpty(t *testing.T) {
	ctx := context.Background()
	logic := NewRefillCommentCacheLogic(ctx, &svc.ServiceContext{})

	if _, err := logic.RefillCommentCache(nil); err == nil {
		t.Fatal("expected nil request error")
	}

	resp, err := logic.RefillCommentCache(&interaction.RefillCommentCacheReq{
		CommentIds: []int64{0, -1},
	})
	if err != nil {
		t.Fatalf("empty ids returned error: %v", err)
	}
	if len(resp.GetComments()) != 0 {
		t.Fatalf("empty ids comments = %v, want empty", resp.GetComments())
	}
}

func TestRefillCommentCacheDBError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("refill repository unavailable")
	logic := NewRefillCommentCacheLogic(ctx, &svc.ServiceContext{})
	logic.commentRepo = &fakeCommentRepository{batchErr: repoErr}

	resp, err := logic.RefillCommentCache(&interaction.RefillCommentCacheReq{
		CommentIds: []int64{401},
	})
	if err == nil {
		t.Fatal("expected RefillCommentCache error")
	}
	if resp != nil {
		t.Fatalf("RefillCommentCache response = %+v, want nil", resp)
	}
}
