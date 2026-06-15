package recommend

import (
	"context"
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

func assertFloatWithin(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()

	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %f, want %f +/- %f", name, got, want, tolerance)
	}
}
