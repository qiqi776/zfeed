package redisx

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
)

func newRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	return store, client
}

func TestZRangeRevWithScoresByFloatCtx(t *testing.T) {
	store, client := newRedis(t)
	store.ZAdd("ranked", 10, "a")
	store.ZAdd("ranked", 30, "c")
	store.ZAdd("ranked", 20, "b")

	pairs, err := ZRangeRevWithScoresByFloatCtx(context.Background(), client, "ranked", 0, 1)
	if err != nil {
		t.Fatalf("ZRangeRevWithScoresByFloatCtx returned error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("len(pairs) = %d, want 2", len(pairs))
	}
	if pairs[0].Key != "c" || pairs[0].Score != 30 || pairs[1].Key != "b" || pairs[1].Score != 20 {
		t.Fatalf("pairs = %+v, want c:30 b:20", pairs)
	}
}

func TestZRangeByScoreWithScoresAndLimitCtx(t *testing.T) {
	store, client := newRedis(t)
	store.ZAdd("by-score", 10, "a")
	store.ZAdd("by-score", 20, "b")
	store.ZAdd("by-score", 30, "c")

	pairs, err := ZRangeByScoreWithScoresAndLimitCtx(context.Background(), client, "by-score", "10", "30", 1, 2)
	if err != nil {
		t.Fatalf("ZRangeByScoreWithScoresAndLimitCtx returned error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("len(pairs) = %d, want 2", len(pairs))
	}
	if pairs[0].Key != "b" || pairs[0].Score != 20 || pairs[1].Key != "c" || pairs[1].Score != 30 {
		t.Fatalf("pairs = %+v, want b:20 c:30", pairs)
	}
}

func TestZRangeRevByScoreWithScoresAndLimitCtx(t *testing.T) {
	store, client := newRedis(t)
	store.ZAdd("rev-score", 10, "a")
	store.ZAdd("rev-score", 20, "b")
	store.ZAdd("rev-score", 30, "c")

	pairs, err := ZRangeRevByScoreWithScoresAndLimitCtx(context.Background(), client, "rev-score", "30", "10", 1, 2)
	if err != nil {
		t.Fatalf("ZRangeRevByScoreWithScoresAndLimitCtx returned error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("len(pairs) = %d, want 2", len(pairs))
	}
	if pairs[0].Key != "b" || pairs[0].Score != 20 || pairs[1].Key != "a" || pairs[1].Score != 10 {
		t.Fatalf("pairs = %+v, want b:20 a:10", pairs)
	}
}
