package recommend

import (
	"context"
	"testing"
	"time"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestLoadPersonalizedSnapshotMetaReadsStoredVariantAndConfigHash(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	snapshotID, err := SavePersonalizedSnapshotWithMeta(
		context.Background(),
		client,
		contentconfig.RecommendConfig{SnapshotTTL: int(time.Minute.Seconds())},
		1001,
		[]Candidate{{ContentID: 6101}},
		SnapshotMeta{
			VariantID:  "b",
			ConfigHash: "hash123",
		},
		time.Unix(1_000, 0),
	)
	if err != nil {
		t.Fatalf("SavePersonalizedSnapshotWithMeta returned error: %v", err)
	}
	if !store.Exists(redisconsts.BuildRecommendUserSnapshotMetaKey(snapshotID)) {
		t.Fatalf("snapshot meta key for %q does not exist", snapshotID)
	}

	got, ok, err := LoadPersonalizedSnapshotMeta(context.Background(), client, snapshotID)
	if err != nil {
		t.Fatalf("LoadPersonalizedSnapshotMeta returned error: %v", err)
	}
	if !ok {
		t.Fatal("LoadPersonalizedSnapshotMeta ok = false, want true")
	}
	if got.VariantID != "b" || got.ConfigHash != "hash123" {
		t.Fatalf("snapshot meta = %+v, want variant b and hash123", got)
	}
}

func TestPersonalizedSnapshotStoresContentSources(t *testing.T) {
	store, client := newRecommendRedis(t)
	defer store.Close()

	snapshotID, err := SavePersonalizedSnapshotWithMeta(
		context.Background(),
		client,
		contentconfig.RecommendConfig{SnapshotTTL: int(time.Minute.Seconds())},
		1001,
		[]Candidate{
			{
				ContentID: 6101,
				SourceScores: map[Source]float64{
					SourceNewContent: 0.9,
					SourceHot:        0.2,
				},
			},
			{
				ContentID: 6102,
				SourceScores: map[Source]float64{
					SourceInterest: 0.7,
				},
			},
		},
		SnapshotMeta{
			VariantID:  "b",
			ConfigHash: "hash123",
		},
		time.Unix(1_000, 0),
	)
	if err != nil {
		t.Fatalf("SavePersonalizedSnapshotWithMeta returned error: %v", err)
	}

	sourceKey := redisconsts.BuildRecommendUserSnapshotSourceKey(snapshotID)
	if !store.Exists(sourceKey) {
		t.Fatalf("snapshot source key for %q does not exist", snapshotID)
	}

	got, err := LoadPersonalizedSnapshotSources(
		context.Background(),
		client,
		snapshotID,
		[]int64{6101, 6102, 6103},
	)
	if err != nil {
		t.Fatalf("LoadPersonalizedSnapshotSources returned error: %v", err)
	}
	want := map[int64]Source{
		6101: SourceNewContent,
		6102: SourceInterest,
	}
	if len(got) != len(want) {
		t.Fatalf("sources = %+v, want %+v", got, want)
	}
	for id, source := range want {
		if got[id] != source {
			t.Fatalf("source[%d] = %q, want %q", id, got[id], source)
		}
	}
}
