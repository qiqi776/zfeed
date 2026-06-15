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
