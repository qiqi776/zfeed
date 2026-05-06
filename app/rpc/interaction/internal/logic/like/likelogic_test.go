package likelogic

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/logx"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/svc"
)

type stubLikeRepo struct {
	isLikedFunc func(userID int64, contentID int64) (bool, error)
}

func (r *stubLikeRepo) Upsert(*do.LikeDO) error {
	return nil
}

func (r *stubLikeRepo) CountByContentID(int64) (int64, error) {
	return 0, nil
}

func (r *stubLikeRepo) CountByContentIDs([]int64) (map[int64]int64, error) {
	return map[int64]int64{}, nil
}

func (r *stubLikeRepo) IsLiked(userID int64, contentID int64) (bool, error) {
	if r.isLikedFunc == nil {
		return false, nil
	}

	return r.isLikedFunc(userID, contentID)
}

func (r *stubLikeRepo) BatchIsLiked(int64, []int64) (map[int64]bool, error) {
	return map[int64]bool{}, nil
}

func TestLikeThenUnlikeStillWorksAfterManyWrites(t *testing.T) {
	t.Parallel()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	svcCtx := &svc.ServiceContext{
		Redis: client,
	}

	likeLogic := NewLikeLogic(context.Background(), svcCtx)
	unlikeLogic := NewUnlikeLogic(context.Background(), svcCtx)

	const (
		userID        int64 = 1001
		firstContent  int64 = 1
		totalContents int64 = 10050
	)

	for contentID := int64(1); contentID <= totalContents; contentID++ {
		changed, err := likeLogic.processLike(userID, contentID)
		if err != nil {
			t.Fatalf("processLike(%d) returned error: %v", contentID, err)
		}
		if !changed {
			t.Fatalf("processLike(%d) changed=false, want true", contentID)
		}
	}

	changed, err := unlikeLogic.processUnlike(userID, firstContent)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if !changed {
		t.Fatalf("processUnlike(%d) changed=false, want true", firstContent)
	}

	changed, err = likeLogic.processLike(userID, firstContent)
	if err != nil {
		t.Fatalf("processLike after unlike returned error: %v", err)
	}
	if !changed {
		t.Fatalf("processLike(%d) after unlike changed=false, want true", firstContent)
	}
}

func TestProcessUnlikeFallsBackToDBWhenCacheExpired(t *testing.T) {
	t.Parallel()

	db := newLikeLogicTestDB(t)
	client := newLikeLogicTestRedis(t)

	if err := db.Create(&likeTestRow{
		UserID:        1001,
		ContentID:     9001,
		ContentUserID: 2001,
		Status:        10,
		IsDeleted:     0,
	}).Error; err != nil {
		t.Fatalf("seed like row: %v", err)
	}

	logic := NewUnlikeLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	changed, err := logic.processUnlike(1001, 9001)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if !changed {
		t.Fatal("processUnlike changed=false, want true when DB still shows liked")
	}
}

func TestProcessUnlikeReturnsFalseWhenCacheExpiredAndDBNotLiked(t *testing.T) {
	t.Parallel()

	db := newLikeLogicTestDB(t)
	client := newLikeLogicTestRedis(t)

	logic := NewUnlikeLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	changed, err := logic.processUnlike(1001, 9001)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if changed {
		t.Fatal("processUnlike changed=true, want false when DB has no liked record")
	}
}

func TestProcessUnlikeSkipsDBWhenCacheShowsAlreadyUnliked(t *testing.T) {
	t.Parallel()

	client := newLikeLogicTestRedis(t)
	if err := cacheLikeState(context.Background(), client, likeCacheKey(1001), "9001", false); err != nil {
		t.Fatalf("seed unlike cache: %v", err)
	}

	logic := &UnlikeLogic{
		ctx:    context.Background(),
		svcCtx: &svc.ServiceContext{Redis: client},
		Logger: logx.WithContext(context.Background()),
		likeRepo: &stubLikeRepo{
			isLikedFunc: func(userID int64, contentID int64) (bool, error) {
				t.Fatalf("unexpected DB fallback, user_id=%d, content_id=%d", userID, contentID)
				return false, nil
			},
		},
	}

	changed, err := logic.processUnlike(1001, 9001)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if changed {
		t.Fatal("processUnlike changed=true, want false when cache already marks unlike")
	}
}
