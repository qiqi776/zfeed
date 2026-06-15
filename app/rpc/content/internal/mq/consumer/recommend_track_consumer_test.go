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

type fakeProfileUpdater struct {
	events []track.Event
	err    error
}

func (u *fakeProfileUpdater) Apply(_ context.Context, event track.Event) error {
	u.events = append(u.events, event)
	return u.err
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

func TestRecommendTrackConsumerNormalizesInteractionLikeEvent(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"event_id":"like_1001_2001_1781480000000000000","event_type":"like","user_id":1001,"content_id":2001,"content_user_id":3001,"scene":"ARTICLE","timestamp":1781480000000000000}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "like_1001_2001_1781480000000000000",
		EventType:  track.EventTypeLike,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerMapsCancelLikeToUnlike(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"event_id":"cancel_like_1001_2001_1781480000000000000","event_type":"cancel_like","user_id":1001,"content_id":2001,"content_user_id":3001,"scene":"ARTICLE","timestamp":1781480000000000000}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "cancel_like_1001_2001_1781480000000000000",
		EventType:  track.EventTypeUnlike,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerNormalizesFavoriteEventRow(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"id":1,"event_id":"favorite_1001_2001_1781480000000000000","event_type":"favorite","scene":1,"user_id":1001,"content_id":2001,"content_user_id":3001}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "favorite_1001_2001_1781480000000000000",
		EventType:  track.EventTypeFavorite,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerMapsRemoveFavoriteToUnfavorite(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"id":2,"event_id":"remove_favorite_1001_2001_1781480000000000000",` +
		`"event_type":"remove_favorite","scene":1,"user_id":1001,"content_id":2001,"content_user_id":3001}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "remove_favorite_1001_2001_1781480000000000000",
		EventType:  track.EventTypeUnfavorite,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerNormalizesUserActionEvent(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"event_id":"ua_1001_2001_1781480000","action":"favorite","user_id":1001,` +
		`"target_type":"content","target_id":2001,"source":"interaction","occurred_at":1781480000}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "ua_1001_2001_1781480000",
		EventType:  track.EventTypeFavorite,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 1781480000,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
	}
}

func TestRecommendTrackConsumerNormalizesCommentRow(t *testing.T) {
	aggregator := &fakeDailyAggregator{}
	updater := &fakeProfileUpdater{}
	consumer := newRecommendTrackConsumerWithProfileForTest(context.Background(), aggregator, updater)
	raw := `{"id":9001,"content_id":2001,"content_user_id":3001,"user_id":1001,` +
		`"comment":"nice post","status":10,"is_deleted":0,"created_at":"1970-01-01T00:00:42Z"}`

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}

	want := track.Event{
		EventID:    "comment_1001_2001_9001",
		EventType:  track.EventTypeComment,
		UserID:     1001,
		ContentID:  2001,
		Source:     "interaction",
		OccurredAt: 42,
	}
	if len(updater.events) != 1 || updater.events[0] != want {
		t.Fatalf("profile events = %+v, want [%+v]", updater.events, want)
	}
	if len(aggregator.events) != 1 || aggregator.events[0] != want {
		t.Fatalf("aggregated events = %+v, want [%+v]", aggregator.events, want)
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
