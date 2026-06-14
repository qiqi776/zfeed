package feedlogic

import (
	"context"
	"testing"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

func TestEmitRecommendTrackEmitsClickAndDwellEvents(t *testing.T) {
	tests := []struct {
		name        string
		req         *contentpb.EmitRecommendTrackReq
		wantType    string
		wantDwellMs int64
	}{
		{
			name: "click",
			req: &contentpb.EmitRecommendTrackReq{
				UserId:     1001,
				EventType:  track.EventTypeClick,
				ContentId:  2001,
				SnapshotId: "rec:0001:b:hash:1",
				VariantId:  "b",
				Source:     "recommend",
				Position:   3,
				OccurredAt: 123456,
			},
			wantType: track.EventTypeClick,
		},
		{
			name: "dwell",
			req: &contentpb.EmitRecommendTrackReq{
				UserId:     1001,
				EventType:  track.EventTypeDwell,
				ContentId:  2001,
				SnapshotId: "rec:0001:b:hash:1",
				VariantId:  "b",
				Source:     "recommend",
				DwellMs:    12_000,
				OccurredAt: 123457,
			},
			wantType:    track.EventTypeDwell,
			wantDwellMs: 12_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer := &fakeRecommendTrackProducer{}
			logic := NewRecommendTrackLogic(context.Background(), &svc.ServiceContext{
				RecommendTrackProducer: producer,
			})

			oldTrack := recordRecommendTrackEmitMetric
			defer func() {
				recordRecommendTrackEmitMetric = oldTrack
			}()
			metrics := map[string]int{}
			recordRecommendTrackEmitMetric = func(eventType, result string) {
				metrics[eventType+":"+result]++
			}

			resp, err := logic.EmitRecommendTrack(tt.req)
			if err != nil {
				t.Fatalf("EmitRecommendTrack returned error: %v", err)
			}
			if resp == nil {
				t.Fatal("EmitRecommendTrack returned nil response")
			}
			if len(producer.events) != 1 {
				t.Fatalf("events = %+v, want one event", producer.events)
			}

			got := producer.events[0]
			if got.EventType != tt.wantType {
				t.Fatalf("event type = %q, want %q", got.EventType, tt.wantType)
			}
			if got.UserID != tt.req.GetUserId() ||
				got.ContentID != tt.req.GetContentId() ||
				got.SnapshotID != tt.req.GetSnapshotId() ||
				got.VariantID != tt.req.GetVariantId() ||
				got.Source != tt.req.GetSource() ||
				got.Position != int(tt.req.GetPosition()) ||
				got.OccurredAt != tt.req.GetOccurredAt() {
				t.Fatalf("event = %+v, request = %+v", got, tt.req)
			}
			if got.DwellMs != tt.wantDwellMs {
				t.Fatalf("dwell_ms = %d, want %d", got.DwellMs, tt.wantDwellMs)
			}
			if got.EventID == "" {
				t.Fatal("event_id is empty")
			}
			if metrics[tt.wantType+":"+recommendResultSuccess] != 1 {
				t.Fatalf("metrics = %#v, want success for %s", metrics, tt.wantType)
			}
		})
	}
}

func TestEmitRecommendTrackRejectsInvalidEvent(t *testing.T) {
	logic := NewRecommendTrackLogic(context.Background(), &svc.ServiceContext{
		RecommendTrackProducer: &fakeRecommendTrackProducer{},
	})

	if _, err := logic.EmitRecommendTrack(&contentpb.EmitRecommendTrackReq{
		UserId:    1001,
		EventType: "share",
		ContentId: 2001,
	}); err == nil {
		t.Fatal("EmitRecommendTrack returned nil error, want invalid event error")
	}
}
