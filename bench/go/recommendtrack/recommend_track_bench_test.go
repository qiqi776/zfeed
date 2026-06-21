package recommendtrack

import (
	"encoding/json"
	"fmt"
	"testing"

	contentpb "zfeed/app/rpc/content/content"

	"google.golang.org/protobuf/proto"
)

var (
	benchmarkTrackRequests []*contentpb.EmitRecommendTrackReq
	benchmarkTrackJSON     []byte
	benchmarkTrackProto    []byte
)

func BenchmarkRecommendTrackBuildExposureRequests(b *testing.B) {
	items := make([]trackFeedItem, 50)
	for i := range items {
		items[i] = trackFeedItem{
			ContentID:  int64(900000 + i),
			Source:     trackSource(i),
			FinalScore: 1000.0 / float64(i+1),
		}
	}

	b.ReportAllocs()
	for b.Loop() {
		benchmarkTrackRequests = buildExposureRequests(10001, "req-bench", "snap-bench", "variant-a", items)
		if len(benchmarkTrackRequests) != len(items) {
			b.Fatalf("requests = %d, want %d", len(benchmarkTrackRequests), len(items))
		}
	}
}

func BenchmarkRecommendTrackJSONMarshal(b *testing.B) {
	req := buildTrackRequest(10001, 900001, 1)

	b.ReportAllocs()
	for b.Loop() {
		var err error
		benchmarkTrackJSON, err = json.Marshal(req)
		if err != nil {
			b.Fatalf("marshal recommend track request: %v", err)
		}
	}
}

func BenchmarkRecommendTrackProtoMarshal(b *testing.B) {
	req := buildTrackRequest(10001, 900001, 1)

	b.ReportAllocs()
	for b.Loop() {
		var err error
		benchmarkTrackProto, err = proto.Marshal(req)
		if err != nil {
			b.Fatalf("marshal recommend track proto: %v", err)
		}
	}
}

type trackFeedItem struct {
	ContentID  int64
	Source     string
	FinalScore float64
}

func buildExposureRequests(
	userID int64,
	requestID string,
	snapshotID string,
	variantID string,
	items []trackFeedItem,
) []*contentpb.EmitRecommendTrackReq {
	requests := make([]*contentpb.EmitRecommendTrackReq, 0, len(items))
	for i, item := range items {
		if item.ContentID <= 0 {
			continue
		}
		requests = append(requests, &contentpb.EmitRecommendTrackReq{
			UserId:     userID,
			EventType:  "exposure",
			ContentId:  item.ContentID,
			RequestId:  requestID,
			SnapshotId: snapshotID,
			VariantId:  variantID,
			Source:     item.Source,
			Position:   int32(i + 1),
			FinalScore: item.FinalScore,
			OccurredAt: 1775553600,
		})
	}
	return requests
}

func buildTrackRequest(userID int64, contentID int64, position int32) *contentpb.EmitRecommendTrackReq {
	return &contentpb.EmitRecommendTrackReq{
		UserId:     userID,
		EventType:  "click",
		ContentId:  contentID,
		RequestId:  "req-bench",
		SnapshotId: "snap-bench",
		VariantId:  "variant-a",
		Source:     "personalized",
		Position:   position,
		FinalScore: 9.875,
		DwellMs:    3200,
		OccurredAt: 1775553600,
	}
}

func trackSource(index int) string {
	switch index % 3 {
	case 0:
		return "personalized"
	case 1:
		return "hot"
	default:
		return fmt.Sprintf("fallback_%d", index%5)
	}
}
