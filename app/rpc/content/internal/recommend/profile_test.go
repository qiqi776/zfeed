package recommend

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestApplyProfileEventDecaysExistingWeight(t *testing.T) {
	store, client := newRecommendRedis(t)
	cfg := contentconfig.RecommendConfig{}
	contentID := int64(2001)
	if err := WriteContentTags(
		context.Background(),
		client,
		cfg,
		contentID,
		map[string]float64{"go": 1},
		1,
	); err != nil {
		t.Fatalf("WriteContentTags returned error: %v", err)
	}

	profileKey := redisconsts.BuildRecommendUserProfileKey(1001)
	oldUpdatedAt := time.Now().Add(-168 * time.Hour).Unix()
	store.HSet(profileKey, "go", "10")
	store.HSet(profileKey, "_updated_at", strconv.FormatInt(oldUpdatedAt, 10))

	if err := ApplyProfileEvent(context.Background(), client, cfg, ProfileEvent{
		EventID:   "evt-decay-1",
		EventType: ActionLike,
		UserID:    1001,
		ContentID: contentID,
	}); err != nil {
		t.Fatalf("ApplyProfileEvent returned error: %v", err)
	}

	gotRaw := store.HGet(profileKey, "go")
	got, err := strconv.ParseFloat(gotRaw, 64)
	if err != nil {
		t.Fatalf("parse profile weight: %v", err)
	}

	elapsedHours := float64(time.Now().Unix()-oldUpdatedAt) / 3600
	want := 10*math.Exp(-elapsedHours/168) + 1
	assertFloatWithin(t, "profile go weight", got, want, 0.01)

	updatedAt, err := strconv.ParseInt(store.HGet(profileKey, "_updated_at"), 10, 64)
	if err != nil {
		t.Fatalf("parse profile updated_at: %v", err)
	}
	if updatedAt <= oldUpdatedAt {
		t.Fatalf("_updated_at = %d, want newer than %d", updatedAt, oldUpdatedAt)
	}
}

func TestLoadUserProfileRequiresEnoughPositiveTags(t *testing.T) {
	store, client := newRecommendRedis(t)
	profileKey := redisconsts.BuildRecommendUserProfileKey(1001)
	store.HSet(profileKey, "go", "1")
	store.HSet(profileKey, "redis", "-0.5")

	got, err := LoadUserProfile(
		context.Background(),
		client,
		1001,
		contentconfig.RecommendInterestConfig{
			MinTags: 2,
			TopTags: 8,
		},
	)
	if err != nil {
		t.Fatalf("LoadUserProfile returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadUserProfile = %#v, want cold-start empty profile with fewer than 2 positive tags", got)
	}
}

func TestApplyProfileEventTrimsProfileTagsToTopWeights(t *testing.T) {
	store, client := newRecommendRedis(t)
	cfg := contentconfig.RecommendConfig{}
	contentID := int64(2002)
	tags := make(map[string]float64, 60)
	for i := 0; i < 60; i++ {
		tags[fmt.Sprintf("tag:%02d", i)] = float64(i + 1)
	}
	if err := WriteContentTags(
		context.Background(),
		client,
		cfg,
		contentID,
		tags,
		1,
	); err != nil {
		t.Fatalf("WriteContentTags returned error: %v", err)
	}

	if err := ApplyProfileEvent(context.Background(), client, cfg, ProfileEvent{
		EventID:   "evt-trim-1",
		EventType: ActionLike,
		UserID:    1001,
		ContentID: contentID,
	}); err != nil {
		t.Fatalf("ApplyProfileEvent returned error: %v", err)
	}

	profileKey := redisconsts.BuildRecommendUserProfileKey(1001)
	raw, err := client.HgetallCtx(context.Background(), profileKey)
	if err != nil {
		t.Fatalf("read profile hash: %v", err)
	}

	tagWeights := parseWeights(raw)
	if len(tagWeights) != 50 {
		t.Fatalf("profile tag count = %d, want 50", len(tagWeights))
	}
	if _, ok := tagWeights["tag:00"]; ok {
		t.Fatal("profile kept lowest-weight tag tag:00, want it trimmed")
	}
	if _, ok := tagWeights["tag:59"]; !ok {
		t.Fatal("profile trimmed highest-weight tag tag:59, want it kept")
	}
	if store.HGet(profileKey, "_updated_at") == "" {
		t.Fatal("profile missing _updated_at after trim")
	}
}

func assertFloatWithin(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()

	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %f, want %f +/- %f", name, got, want, tolerance)
	}
}
