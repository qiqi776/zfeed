package feedlogic

import (
	"context"
	"strconv"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

func TestEmitRecommendTrackEmitsClientEvents(t *testing.T) {
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
		{
			name: "like",
			req: &contentpb.EmitRecommendTrackReq{
				UserId:     1001,
				EventType:  track.EventTypeLike,
				ContentId:  2001,
				SnapshotId: "rec:0001:b:hash:1",
				VariantId:  "b",
				Source:     "recommend",
				Position:   4,
				OccurredAt: 123458,
			},
			wantType: track.EventTypeLike,
		},
		{
			name: "favorite",
			req: &contentpb.EmitRecommendTrackReq{
				UserId:     1001,
				EventType:  track.EventTypeFavorite,
				ContentId:  2001,
				SnapshotId: "rec:0001:b:hash:1",
				VariantId:  "b",
				Source:     "recommend",
				Position:   5,
				OccurredAt: 123459,
			},
			wantType: track.EventTypeFavorite,
		},
		{
			name: "comment",
			req: &contentpb.EmitRecommendTrackReq{
				UserId:     1001,
				EventType:  track.EventTypeComment,
				ContentId:  2001,
				SnapshotId: "rec:0001:b:hash:1",
				VariantId:  "b",
				Source:     "recommend",
				Position:   6,
				OccurredAt: 123460,
			},
			wantType: track.EventTypeComment,
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

func TestEmitRecommendTrackUpdatesUserProfileAfterSuccessfulEmit(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		wantGo    float64
	}{
		{name: "click", eventType: track.EventTypeClick, wantGo: 0.5},
		{name: "dwell", eventType: track.EventTypeDwell, wantGo: 0.8},
		{name: "like", eventType: track.EventTypeLike, wantGo: 1},
		{name: "favorite", eventType: track.EventTypeFavorite, wantGo: 3},
		{name: "comment", eventType: track.EventTypeComment, wantGo: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				map[string]float64{"go": 1, "redis": 0.5},
				1,
			); err != nil {
				t.Fatalf("WriteContentTags returned error: %v", err)
			}

			logic := NewRecommendTrackLogic(context.Background(), &svc.ServiceContext{
				Config: contentconfig.Config{
					Recommend: cfg,
				},
				Redis:                  redisClient,
				RecommendTrackProducer: &fakeRecommendTrackProducer{},
			})

			req := &contentpb.EmitRecommendTrackReq{
				UserId:     1001,
				EventType:  tt.eventType,
				ContentId:  contentID,
				SnapshotId: "rec:0001:b:hash:1",
				VariantId:  "b",
				Source:     "recommend",
				Position:   3,
				OccurredAt: 123456,
			}
			if tt.eventType == track.EventTypeDwell {
				req.DwellMs = 12_000
			}

			if _, err := logic.EmitRecommendTrack(req); err != nil {
				t.Fatalf("EmitRecommendTrack returned error: %v", err)
			}

			profileKey := redisconsts.BuildRecommendUserProfileKey(1001)
			gotRaw := store.HGet(profileKey, "go")
			got, err := strconv.ParseFloat(gotRaw, 64)
			if err != nil {
				t.Fatalf("parse profile tag: %v", err)
			}
			if got != tt.wantGo {
				t.Fatalf("profile go weight = %v, want %v", got, tt.wantGo)
			}
		})
	}
}
