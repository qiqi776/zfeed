package recommend

import (
	"context"
	"math"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestNormalizeConfigFillsRecommendDefaults(t *testing.T) {
	got := NormalizeConfig(contentconfig.RecommendConfig{})

	if got.TimeoutMs != 120 {
		t.Fatalf("TimeoutMs = %d, want 120", got.TimeoutMs)
	}
	if got.SnapshotTTL != 600 {
		t.Fatalf("SnapshotTTL = %d, want 600", got.SnapshotTTL)
	}
	if got.CandidateTTL != 300 {
		t.Fatalf("CandidateTTL = %d, want 300", got.CandidateTTL)
	}
	if !got.Hot.Enabled || got.Hot.Weight != 0.55 || got.Hot.Limit != 300 {
		t.Fatalf("Hot defaults = %+v, want enabled weight=0.55 limit=300", got.Hot)
	}
	if got.Rank.AlphaHot != 0.45 ||
		got.Rank.BetaInterest != 0.30 ||
		got.Rank.GammaFresh != 0.20 ||
		got.Rank.DeltaQuality != 0.05 ||
		got.Rank.SeenPenalty != 0.30 {
		t.Fatalf("Rank defaults = %+v, want alpha/beta/gamma/delta/penalty 0.45/0.30/0.20/0.05/0.30", got.Rank)
	}
}

func TestLoadRuntimeConfigMergesRedisOverrides(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "enabled", "false"); err != nil {
		t.Fatalf("hset enabled: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "fallback_to_hot", "false"); err != nil {
		t.Fatalf("hset fallback_to_hot: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.hot.weight", "0.70"); err != nil {
		t.Fatalf("hset hot weight: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.hot.enabled", "false"); err != nil {
		t.Fatalf("hset hot enabled: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.hot.limit", "77"); err != nil {
		t.Fatalf("hset hot limit: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.new_content.enabled", "false"); err != nil {
		t.Fatalf("hset new enabled: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.new_content.limit", "88"); err != nil {
		t.Fatalf("hset new limit: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.new_content.weight", "invalid"); err != nil {
		t.Fatalf("hset new weight: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.interest.enabled", "true"); err != nil {
		t.Fatalf("hset interest enabled: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.interest.limit", "99"); err != nil {
		t.Fatalf("hset interest limit: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "recall.interest.weight", "0.40"); err != nil {
		t.Fatalf("hset interest weight: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "rank.coarse_limit", "111"); err != nil {
		t.Fatalf("hset coarse limit: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "rank.alpha_hot", "invalid"); err != nil {
		t.Fatalf("hset alpha: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "rank.beta_interest", "0.60"); err != nil {
		t.Fatalf("hset beta: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "rank.gamma_fresh", "0.10"); err != nil {
		t.Fatalf("hset gamma: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "diversity.author_window", "9"); err != nil {
		t.Fatalf("hset author window: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "diversity.enabled", "false"); err != nil {
		t.Fatalf("hset diversity enabled: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "diversity.max_same_author", "2"); err != nil {
		t.Fatalf("hset max same author: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "diversity.type_window", "8"); err != nil {
		t.Fatalf("hset type window: %v", err)
	}
	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "diversity.max_same_type", "3"); err != nil {
		t.Fatalf("hset max same type: %v", err)
	}

	got, err := LoadRuntimeConfig(context.Background(), client, contentconfig.RecommendConfig{
		Enabled:       true,
		FallbackToHot: true,
		NewContent: contentconfig.RecommendNewContentConfig{
			Weight: 0.20,
		},
	})
	if err != nil {
		t.Fatalf("LoadRuntimeConfig returned error: %v", err)
	}

	if got.Enabled {
		t.Fatal("Enabled = true, want false from Redis override")
	}
	if got.FallbackToHot {
		t.Fatal("FallbackToHot = true, want false from Redis override")
	}
	if got.Hot.Enabled {
		t.Fatal("Hot.Enabled = true, want false from Redis override")
	}
	if got.Hot.Limit != 77 {
		t.Fatalf("Hot.Limit = %d, want 77", got.Hot.Limit)
	}
	if got.NewContent.Enabled {
		t.Fatal("NewContent.Enabled = true, want false from Redis override")
	}
	if got.NewContent.Limit != 88 {
		t.Fatalf("NewContent.Limit = %d, want 88", got.NewContent.Limit)
	}
	if !got.Interest.Enabled {
		t.Fatal("Interest.Enabled = false, want true from Redis override")
	}
	if got.Interest.Limit != 99 {
		t.Fatalf("Interest.Limit = %d, want 99", got.Interest.Limit)
	}
	if got.Rank.CoarseLimit != 111 {
		t.Fatalf("Rank.CoarseLimit = %d, want 111", got.Rank.CoarseLimit)
	}
	if got.Diversity.Enabled {
		t.Fatal("Diversity.Enabled = true, want false from Redis override")
	}
	assertFloat(t, "Hot.Weight", got.Hot.Weight, 0.70)
	assertFloat(t, "NewContent.Weight", got.NewContent.Weight, 0.20)
	assertFloat(t, "Interest.Weight", got.Interest.Weight, 0.40)
	assertFloat(t, "Rank.AlphaHot", got.Rank.AlphaHot, 0.45)
	assertFloat(t, "Rank.BetaInterest", got.Rank.BetaInterest, 0.60)
	assertFloat(t, "Rank.GammaFresh", got.Rank.GammaFresh, 0.10)
	if got.Diversity.AuthorWindow != 9 {
		t.Fatalf("Diversity.AuthorWindow = %d, want 9", got.Diversity.AuthorWindow)
	}
	if got.Diversity.MaxSameAuthor != 2 {
		t.Fatalf("Diversity.MaxSameAuthor = %d, want 2", got.Diversity.MaxSameAuthor)
	}
	if got.Diversity.TypeWindow != 8 {
		t.Fatalf("Diversity.TypeWindow = %d, want 8", got.Diversity.TypeWindow)
	}
	if got.Diversity.MaxSameType != 3 {
		t.Fatalf("Diversity.MaxSameType = %d, want 3", got.Diversity.MaxSameType)
	}
}

func TestLoadRuntimeConfigWithoutRedisReturnsNormalizedBase(t *testing.T) {
	got, err := LoadRuntimeConfig(context.Background(), nil, contentconfig.RecommendConfig{})
	if err != nil {
		t.Fatalf("LoadRuntimeConfig returned error: %v", err)
	}
	if got.TimeoutMs != 120 || got.Rank.AlphaHot != 0.45 {
		t.Fatalf("config = %+v, want normalized defaults", got)
	}
}

func TestLoadRuntimeConfigUsesTenSecondCache(t *testing.T) {
	resetRuntimeConfigCacheForTest()
	t.Cleanup(resetRuntimeConfigCacheForTest)

	store, client := newRecommendRedis(t)
	defer store.Close()

	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "enabled", "false"); err != nil {
		t.Fatalf("hset enabled: %v", err)
	}

	base := contentconfig.RecommendConfig{Enabled: true}
	clock := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	oldNow := runtimeConfigNow
	runtimeConfigNow = func() time.Time { return clock }
	defer func() { runtimeConfigNow = oldNow }()

	got1, err := LoadRuntimeConfig(context.Background(), client, base)
	if err != nil {
		t.Fatalf("first LoadRuntimeConfig returned error: %v", err)
	}
	if got1.Enabled {
		t.Fatal("first load Enabled = true, want false from Redis")
	}

	if err := client.HsetCtx(context.Background(), RuntimeFlagKey, "enabled", "true"); err != nil {
		t.Fatalf("hset enabled true: %v", err)
	}

	clock = clock.Add(5 * time.Second)
	got2, err := LoadRuntimeConfig(context.Background(), client, base)
	if err != nil {
		t.Fatalf("second LoadRuntimeConfig returned error: %v", err)
	}
	if got2.Enabled {
		t.Fatal("second load Enabled = true, want cached false value within 10s")
	}

	clock = clock.Add(6 * time.Second)
	got3, err := LoadRuntimeConfig(context.Background(), client, base)
	if err != nil {
		t.Fatalf("third LoadRuntimeConfig returned error: %v", err)
	}
	if !got3.Enabled {
		t.Fatal("third load Enabled = false, want refreshed true value after cache expiry")
	}
}

func newRecommendRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	return store, client
}

func assertFloat(t *testing.T, name string, got float64, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("%s = %f, want %f", name, got, want)
	}
}
