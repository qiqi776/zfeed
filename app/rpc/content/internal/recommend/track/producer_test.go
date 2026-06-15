package track

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestIsClientEventTypeAllowsInteractionEvents(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      bool
	}{
		{name: "click", eventType: EventTypeClick, want: true},
		{name: "dwell", eventType: EventTypeDwell, want: true},
		{name: "like", eventType: EventTypeLike, want: true},
		{name: "favorite", eventType: EventTypeFavorite, want: true},
		{name: "comment", eventType: EventTypeComment, want: true},
		{name: "unlike", eventType: "unlike", want: true},
		{name: "unfavorite", eventType: EventTypeUnfavorite, want: true},
		{name: "exposure is server side", eventType: EventTypeExposure, want: false},
		{name: "unknown", eventType: "share", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsClientEventType(tt.eventType); got != tt.want {
				t.Fatalf("IsClientEventType(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

type fakeMessagePusher struct {
	payloads []string
	err      error
}

func (p *fakeMessagePusher) Push(_ context.Context, value string) error {
	p.payloads = append(p.payloads, value)
	return p.err
}

func TestKafkaProducerEmitMarshalsEvent(t *testing.T) {
	pusher := &fakeMessagePusher{}
	producer := NewKafkaProducer(pusher, 1)

	event := Event{
		EventID:    "rec_exposure_1_2_3",
		EventType:  EventTypeExposure,
		UserID:     1001,
		ContentID:  2002,
		SnapshotID: "rec:0001:control:hash:1",
		VariantID:  "control",
		Source:     "recommend",
		Position:   1,
		OccurredAt: 1234567890,
	}

	if err := producer.Emit(context.Background(), event); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if len(pusher.payloads) != 1 {
		t.Fatalf("payloads = %v, want 1 event", pusher.payloads)
	}

	var got Event
	if err := json.Unmarshal([]byte(pusher.payloads[0]), &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got != event {
		t.Fatalf("event = %+v, want %+v", got, event)
	}
}

func TestKafkaProducerEmitReturnsPushError(t *testing.T) {
	pusher := &fakeMessagePusher{err: errors.New("kafka down")}
	producer := NewKafkaProducer(pusher, 1)

	if err := producer.Emit(context.Background(), Event{EventID: "1", EventType: EventTypeExposure}); err == nil {
		t.Fatal("Emit returned nil error, want push error")
	}
}
