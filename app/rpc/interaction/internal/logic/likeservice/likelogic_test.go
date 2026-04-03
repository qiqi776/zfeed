package likeservicelogic

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/rpc/interaction/internal/svc"
)

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
}
