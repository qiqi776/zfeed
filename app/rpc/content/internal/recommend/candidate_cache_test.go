package recommend

import (
	"context"
	"testing"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestBuildCandidateCacheKeyUsesBucketVariantAndConfigHash(t *testing.T) {
	got := BuildCandidateCacheKey(10001, " b ", " hash123 ")
	want := redisconsts.RecommendCandidatePrefix + ":0001:b:hash123"
	if got != want {
		t.Fatalf("candidate cache key = %q, want %q", got, want)
	}
}

func TestCandidateCacheSaveAndLoad(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	key := BuildCandidateCacheKey(42, "control", "hashabc")
	cfg := contentconfig.RecommendConfig{CandidateTTL: 120}
	candidates := []Candidate{
		{ContentID: 1001, Score: 0.8},
		{ContentID: 1002, Score: 1.2},
		{ContentID: 0, Score: 9},
	}

	if err := SaveCandidateCache(context.Background(), client, cfg, key, candidates); err != nil {
		t.Fatalf("SaveCandidateCache returned error: %v", err)
	}

	if !store.Exists(key) {
		t.Fatalf("candidate cache key %q does not exist", key)
	}
	if ttl := store.TTL(key); ttl <= 0 || ttl > 120*1_000_000_000 {
		t.Fatalf("candidate cache ttl = %s, want within 120s", ttl)
	}

	got, ok, err := LoadCandidateCache(context.Background(), client, key, 10)
	if err != nil {
		t.Fatalf("LoadCandidateCache returned error: %v", err)
	}
	if !ok {
		t.Fatal("LoadCandidateCache ok = false, want true")
	}
	wantIDs := []int64{1002, 1001}
	if len(got) != len(wantIDs) {
		t.Fatalf("len(candidates) = %d, want %d: %+v", len(got), len(wantIDs), got)
	}
	for i, wantID := range wantIDs {
		if got[i].ContentID != wantID {
			t.Fatalf("candidate ids = %+v, want %v", got, wantIDs)
		}
	}
	if got[0].Score != 1.2 || got[1].Score != 0.8 {
		t.Fatalf("candidate scores = [%f %f], want [1.2 0.8]", got[0].Score, got[1].Score)
	}
}

func TestCandidateCachePreservesPrimarySources(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	key := BuildCandidateCacheKey(42, "control", "hashabc")
	cfg := contentconfig.RecommendConfig{CandidateTTL: 120}
	candidates := []Candidate{
		{
			ContentID: 1001,
			Score:     0.8,
			SourceScores: map[Source]float64{
				SourceHot:        0.2,
				SourceNewContent: 0.9,
			},
		},
		{
			ContentID: 1002,
			Score:     1.2,
			SourceScores: map[Source]float64{
				SourceInterest: 0.7,
			},
		},
	}

	if err := SaveCandidateCache(context.Background(), client, cfg, key, candidates); err != nil {
		t.Fatalf("SaveCandidateCache returned error: %v", err)
	}

	got, ok, err := LoadCandidateCache(context.Background(), client, key, 10)
	if err != nil {
		t.Fatalf("LoadCandidateCache returned error: %v", err)
	}
	if !ok {
		t.Fatal("LoadCandidateCache ok = false, want true")
	}
	wantSources := map[int64]Source{
		1001: SourceNewContent,
		1002: SourceInterest,
	}
	for _, candidate := range got {
		if PrimarySource(candidate) != wantSources[candidate.ContentID] {
			t.Fatalf(
				"candidate %d primary source = %q, want %q",
				candidate.ContentID,
				PrimarySource(candidate),
				wantSources[candidate.ContentID],
			)
		}
	}
}

func TestLoadCandidateCacheMiss(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	got, ok, err := LoadCandidateCache(context.Background(), client, "missing", 10)
	if err != nil {
		t.Fatalf("LoadCandidateCache returned error: %v", err)
	}
	if ok {
		t.Fatalf("ok = true, want false with candidates %+v", got)
	}
}
