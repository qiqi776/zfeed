package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"zfeed/app/rpc/content/internal/recommend/track"
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
