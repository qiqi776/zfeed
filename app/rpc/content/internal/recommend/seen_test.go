package recommend

import (
	"context"
	"testing"
	"time"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestRecordSeenContentsWritesRecentExposureSet(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	cfg := contentconfig.RecommendConfig{SeenTTL: 3600}
	now := time.Unix(1780191000, 0)
	if err := RecordSeenContents(context.Background(), client, cfg, 1001, []int64{9001, 9002, 0}, now); err != nil {
		t.Fatalf("RecordSeenContents returned error: %v", err)
	}

	key := redisconsts.BuildRecommendSeenKey(1001)
	if !store.Exists(key) {
		t.Fatalf("seen key %q does not exist", key)
	}
	got, err := store.ZScore(key, "9001")
	if err != nil {
		t.Fatalf("zscore seen member: %v", err)
	}
	if got != float64(now.Unix()) {
		t.Fatalf("seen score = %f, want %d", got, now.Unix())
	}
	if store.Exists("0") {
		t.Fatal("invalid content id was written")
	}
	if ttl := store.TTL(key); ttl <= 0 || ttl > time.Hour {
		t.Fatalf("seen ttl = %s, want within 1h", ttl)
	}
}

func TestLoadSeenCountsReturnsOnlyRequestedIDs(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	key := redisconsts.BuildRecommendSeenKey(1001)
	now := time.Now().Unix()
	store.ZAdd(key, float64(now), "9001")
	store.ZAdd(key, float64(now), "9002")
	store.ZAdd(key, float64(now), "9003")

	got, err := LoadSeenCounts(context.Background(), client, 1001, []int64{9001, 9002, 9999, 0})
	if err != nil {
		t.Fatalf("LoadSeenCounts returned error: %v", err)
	}
	if got[9001] != 1 || got[9002] != 1 {
		t.Fatalf("seen counts = %#v, want 9001/9002 seen once", got)
	}
	if got[9999] != 0 {
		t.Fatalf("seen count for missing id = %d, want 0", got[9999])
	}
	if _, ok := got[9003]; ok {
		t.Fatalf("seen counts includes unrequested id 9003: %#v", got)
	}
}

func TestRecordSeenContentsAccumulatesExposureCounts(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	cfg := contentconfig.RecommendConfig{SeenTTL: 3600}
	now := time.Now()
	userID := int64(1001)
	if err := RecordSeenContents(
		context.Background(),
		client,
		cfg,
		userID,
		[]int64{9001, 9002},
		now.Add(-time.Minute),
	); err != nil {
		t.Fatalf("first RecordSeenContents returned error: %v", err)
	}
	if err := RecordSeenContents(
		context.Background(),
		client,
		cfg,
		userID,
		[]int64{9001},
		now,
	); err != nil {
		t.Fatalf("second RecordSeenContents returned error: %v", err)
	}

	got, err := LoadSeenCounts(context.Background(), client, userID, []int64{9001, 9002})
	if err != nil {
		t.Fatalf("LoadSeenCounts returned error: %v", err)
	}
	if got[9001] != 2 {
		t.Fatalf("seen count for repeated exposure = %d, want 2", got[9001])
	}
	if got[9002] != 1 {
		t.Fatalf("seen count for single exposure = %d, want 1", got[9002])
	}

	seenKey := redisconsts.BuildRecommendSeenKey(userID)
	if score, err := store.ZScore(seenKey, "9001"); err != nil || score != float64(now.Unix()) {
		t.Fatalf("seen zset score = %f, err=%v, want latest exposure unix", score, err)
	}
}

func TestLoadSeenCountsUsesExactRollingTwentyFourHourHistory(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	now := time.Now()
	userID := int64(1001)
	historyKey := redisconsts.BuildRecommendSeenHistoryKey(userID, 9001)
	store.ZAdd(historyKey, float64(now.Add(-24*time.Hour+10*time.Minute).Unix()), "inside-window")
	store.ZAdd(historyKey, float64(now.Unix()), "fresh")
	store.ZAdd(historyKey, float64(now.Add(-24*time.Hour-10*time.Minute).Unix()), "expired")

	got, err := LoadSeenCounts(context.Background(), client, userID, []int64{9001})
	if err != nil {
		t.Fatalf("LoadSeenCounts returned error: %v", err)
	}
	if got[9001] != 2 {
		t.Fatalf("rolling 24h seen count = %d, want 2", got[9001])
	}
}

func TestLoadSeenCountsIgnoresLegacyCountWhenLastSeenExpired(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	now := time.Now()
	userID := int64(1001)
	contentID := int64(9001)
	store.HSet(redisconsts.BuildRecommendSeenCountKey(userID), "9001", "3")
	store.ZAdd(redisconsts.BuildRecommendSeenKey(userID), float64(now.Add(-25*time.Hour).Unix()), "9001")

	got, err := LoadSeenCounts(context.Background(), client, userID, []int64{contentID})
	if err != nil {
		t.Fatalf("LoadSeenCounts returned error: %v", err)
	}
	if got[contentID] != 0 {
		t.Fatalf("legacy expired seen count = %d, want 0", got[contentID])
	}
}

func TestLoadSeenCountsIgnoresExpiredDailyExposureCount(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	cfg := contentconfig.RecommendConfig{SeenTTL: 3600}
	now := time.Now()
	userID := int64(1001)
	if err := RecordSeenContents(
		context.Background(),
		client,
		cfg,
		userID,
		[]int64{9001},
		now.Add(-25*time.Hour),
	); err != nil {
		t.Fatalf("old RecordSeenContents returned error: %v", err)
	}
	if err := RecordSeenContents(
		context.Background(),
		client,
		cfg,
		userID,
		[]int64{9001},
		now,
	); err != nil {
		t.Fatalf("fresh RecordSeenContents returned error: %v", err)
	}

	got, err := LoadSeenCounts(context.Background(), client, userID, []int64{9001})
	if err != nil {
		t.Fatalf("LoadSeenCounts returned error: %v", err)
	}
	if got[9001] != 1 {
		t.Fatalf("seen count after daily window rollover = %d, want 1", got[9001])
	}
}
