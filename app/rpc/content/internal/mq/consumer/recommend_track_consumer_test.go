package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

type fakeDailyAggregator struct {
	events []track.Event
	err    error
}

func (a *fakeDailyAggregator) Aggregate(_ context.Context, event track.Event) error {
	a.events = append(a.events, event)
	return a.err
}

func TestRecommendTrackConsumerAggregatesTrackEvent(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	consumer := newRecommendTrackConsumerForTest(context.Background(), aggregator)

	event := track.Event{
		EventID:    "rec_click_1001_2001_1",
		EventType:  track.EventTypeClick,
		UserID:     1001,
		ContentID:  2001,
		SnapshotID: "rec:0001:b:hash:1",
		VariantID:  "b",
		Source:     "recommend",
		OccurredAt: 1781480000,
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if err := consumer.Consume(context.Background(), "", string(raw)); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	if len(aggregator.events) != 1 {
		t.Fatalf("aggregated events = %+v, want one event", aggregator.events)
	}
	if aggregator.events[0] != event {
		t.Fatalf("aggregated event = %+v, want %+v", aggregator.events[0], event)
	}
}

func TestRecommendTrackConsumerAppliesProfileEvent(t *testing.T) {
	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	cfg := contentconfig.RecommendConfig{}
	contentID := int64(2001)
	if err := recommend.WriteContentTags(
		context.Background(),
		redisClient,
		cfg,
		contentID,
		map[string]float64{"go": 1},
		1,
	); err != nil {
		t.Fatalf("WriteContentTags returned error: %v", err)
	}

	consumer := NewRecommendTrackConsumer(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: cfg,
		},
		Redis: redisClient,
	})

	event := track.Event{
		EventID:    "rec_like_1001_2001_1",
		EventType:  track.EventTypeLike,
		UserID:     1001,
		ContentID:  contentID,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	for range 2 {
		if err := consumer.Consume(context.Background(), "", string(raw)); err != nil {
			t.Fatalf("Consume returned error: %v", err)
		}
	}

	gotRaw := store.HGet(redisconsts.BuildRecommendUserProfileKey(1001), "go")
	got, err := strconv.ParseFloat(gotRaw, 64)
	if err != nil {
		t.Fatalf("parse profile tag: %v", err)
	}
	if got != 1 {
		t.Fatalf("profile go weight = %v, want 1 after idempotent replay", got)
	}
}

func TestRecommendTrackConsumerRejectsInvalidJSON(t *testing.T) {
	consumer := newRecommendTrackConsumerForTest(context.Background(), &fakeDailyAggregator{})

	if err := consumer.Consume(context.Background(), "", "{"); err == nil {
		t.Fatal("Consume returned nil error, want JSON error")
	}
}

func TestRecommendTrackConsumerReturnsAggregatorError(t *testing.T) {
	wantErr := errors.New("db down")
	consumer := newRecommendTrackConsumerForTest(context.Background(), &fakeDailyAggregator{err: wantErr})

	raw, err := json.Marshal(track.Event{
		EventID:    "rec_click_1001_2001_1",
		EventType:  track.EventTypeClick,
		ContentID:  2001,
		OccurredAt: 1781480000,
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if err := consumer.Consume(context.Background(), "", string(raw)); !errors.Is(err, wantErr) {
		t.Fatalf("Consume error = %v, want %v", err, wantErr)
	}
}
