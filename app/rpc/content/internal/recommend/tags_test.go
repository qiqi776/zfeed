package recommend

import (
	"context"
	"testing"
	"time"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestRefreshContentTagIndexRewritesScoresAndKeepsTTL(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	cfg := contentconfig.RecommendConfig{
		Interest: contentconfig.RecommendInterestConfig{
			ContentTagTTL: 300,
			TagIndexTTL:   120,
		},
	}
	contentID := int64(2001)
	contentKey := redisconsts.BuildRecommendContentTagsKey(contentID)
	if err := client.HsetCtx(context.Background(), contentKey, "go", "0.500000"); err != nil {
		t.Fatalf("seed content tag: %v", err)
	}
	if err := client.HsetCtx(context.Background(), contentKey, "redis", "0.250000"); err != nil {
		t.Fatalf("seed content tag: %v", err)
	}

	refreshed, err := RefreshContentTagIndex(
		context.Background(),
		client,
		cfg,
		contentID,
		20,
	)
	if err != nil {
		t.Fatalf("RefreshContentTagIndex returned error: %v", err)
	}
	if !refreshed {
		t.Fatal("RefreshContentTagIndex refreshed = false, want true")
	}

	member := "2001"
	goScore, err := store.ZScore(redisconsts.BuildRecommendTagIndexKey("go"), member)
	if err != nil {
		t.Fatalf("go tag index score: %v", err)
	}
	if goScore != 10 {
		t.Fatalf("go tag index score = %v, want 10", goScore)
	}
	redisScore, err := store.ZScore(redisconsts.BuildRecommendTagIndexKey("redis"), member)
	if err != nil {
		t.Fatalf("redis tag index score: %v", err)
	}
	if redisScore != 5 {
		t.Fatalf("redis tag index score = %v, want 5", redisScore)
	}
	if ttl := store.TTL(redisconsts.BuildRecommendTagIndexKey("go")); ttl <= 0 || ttl > 120*time.Second {
		t.Fatalf("go tag index ttl = %s, want within 120s", ttl)
	}
}

func TestRefreshContentTagIndexSkipsContentWithoutTags(t *testing.T) {
	_, client := newRecommendRedis(t)

	refreshed, err := RefreshContentTagIndex(
		context.Background(),
		client,
		contentconfig.RecommendConfig{},
		404,
		20,
	)
	if err != nil {
		t.Fatalf("RefreshContentTagIndex returned error: %v", err)
	}
	if refreshed {
		t.Fatal("RefreshContentTagIndex refreshed = true, want false without content tags")
	}
}
