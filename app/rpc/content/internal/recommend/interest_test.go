package recommend

import (
	"context"
	"reflect"
	"testing"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestRecallInterestUsesTagIndexScores(t *testing.T) {
	store, client := newRecommendRedis(t)
	profileKey := redisconsts.BuildRecommendUserProfileKey(1001)
	store.HSet(profileKey, "go", "1")
	store.HSet(profileKey, "redis", "1")

	if _, err := client.ZaddFloatCtx(
		context.Background(),
		redisconsts.BuildRecommendTagIndexKey("go"),
		100,
		"2001",
	); err != nil {
		t.Fatalf("zadd go 2001: %v", err)
	}
	if _, err := client.ZaddFloatCtx(
		context.Background(),
		redisconsts.BuildRecommendTagIndexKey("go"),
		99,
		"2002",
	); err != nil {
		t.Fatalf("zadd go 2002: %v", err)
	}
	if _, err := client.ZaddFloatCtx(
		context.Background(),
		redisconsts.BuildRecommendTagIndexKey("redis"),
		0.1,
		"2002",
	); err != nil {
		t.Fatalf("zadd redis 2002: %v", err)
	}

	got, err := RecallInterest(
		context.Background(),
		client,
		1001,
		contentconfig.RecommendInterestConfig{
			Enabled: true,
			Limit:   2,
			MinTags: 2,
			TopTags: 2,
		},
	)
	if err != nil {
		t.Fatalf("RecallInterest returned error: %v", err)
	}

	want := []int64{2001, 2002}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RecallInterest = %#v, want %#v", got, want)
	}
}

func TestRecallInterestCandidatesPreservesScoresAndRanks(t *testing.T) {
	store, client := newRecommendRedis(t)
	profileKey := redisconsts.BuildRecommendUserProfileKey(1001)
	store.HSet(profileKey, "go", "1")

	if _, err := client.ZaddFloatCtx(
		context.Background(),
		redisconsts.BuildRecommendTagIndexKey("go"),
		100,
		"2001",
	); err != nil {
		t.Fatalf("zadd go 2001: %v", err)
	}
	if _, err := client.ZaddFloatCtx(
		context.Background(),
		redisconsts.BuildRecommendTagIndexKey("go"),
		50,
		"2002",
	); err != nil {
		t.Fatalf("zadd go 2002: %v", err)
	}

	got, err := RecallInterestCandidates(
		context.Background(),
		client,
		1001,
		contentconfig.RecommendInterestConfig{
			Enabled: true,
			Limit:   2,
			MinTags: 1,
			TopTags: 1,
		},
	)
	if err != nil {
		t.Fatalf("RecallInterestCandidates returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ContentID != 2001 || got[1].ContentID != 2002 {
		t.Fatalf("ids = [%d %d], want [2001 2002]", got[0].ContentID, got[1].ContentID)
	}
	if got[0].SourceScores[SourceInterest] != 1 {
		t.Fatalf("first interest source score = %v, want 1", got[0].SourceScores[SourceInterest])
	}
	if got[1].SourceScores[SourceInterest] != 0.5 {
		t.Fatalf("second interest source score = %v, want 0.5", got[1].SourceScores[SourceInterest])
	}
	if got[0].SourceRanks[SourceInterest] != 1 || got[1].SourceRanks[SourceInterest] != 2 {
		t.Fatalf("source ranks = %#v/%#v, want 1/2", got[0].SourceRanks, got[1].SourceRanks)
	}
}
