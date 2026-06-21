package feed

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
)

type fakeFeedRPC struct {
	trackReq *contentpb.EmitRecommendTrackReq
	trackErr error
}

func (f *fakeFeedRPC) RecommendFeed(
	context.Context,
	*contentpb.RecommendFeedReq,
	...grpc.CallOption,
) (*contentpb.RecommendFeedRes, error) {
	return &contentpb.RecommendFeedRes{}, nil
}

func (f *fakeFeedRPC) EmitRecommendTrack(
	_ context.Context,
	in *contentpb.EmitRecommendTrackReq,
	_ ...grpc.CallOption,
) (*contentpb.EmitRecommendTrackRes, error) {
	f.trackReq = in
	return &contentpb.EmitRecommendTrackRes{}, f.trackErr
}

func (f *fakeFeedRPC) FollowFeed(
	context.Context,
	*contentpb.FollowFeedReq,
	...grpc.CallOption,
) (*contentpb.FollowFeedRes, error) {
	return &contentpb.FollowFeedRes{}, nil
}

func (f *fakeFeedRPC) UserPublishFeed(
	context.Context,
	*contentpb.UserPublishFeedReq,
	...grpc.CallOption,
) (*contentpb.UserPublishFeedRes, error) {
	return &contentpb.UserPublishFeedRes{}, nil
}

func (f *fakeFeedRPC) UserFavoriteFeed(
	context.Context,
	*contentpb.UserFavoriteFeedReq,
	...grpc.CallOption,
) (*contentpb.UserFavoriteFeedRes, error) {
	return &contentpb.UserFavoriteFeedRes{}, nil
}

func TestEmitRecommendTrackMaps(t *testing.T) {
	eventType := "click"
	contentID := int64(2001)
	requestID := "req-001"
	snapshotID := "rec:0001:b:hash:1"
	variantID := "b"
	source := "recommend"
	position := int64(3)
	finalScore := 0.82
	dwellMs := int64(12000)
	occurredAt := int64(123456)
	feedRPC := &fakeFeedRPC{}
	logic := NewEmitRecommendTrackLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.EmitRecommendTrack(&types.RecommendTrackReq{
		EventType:  &eventType,
		ContentId:  &contentID,
		RequestId:  &requestID,
		SnapshotId: &snapshotID,
		VariantId:  &variantID,
		Source:     &source,
		Position:   &position,
		FinalScore: &finalScore,
		DwellMs:    &dwellMs,
		OccurredAt: &occurredAt,
	})
	if err != nil {
		t.Fatalf("EmitRecommendTrack returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("EmitRecommendTrack returned nil response")
	}
	if feedRPC.trackReq == nil {
		t.Fatal("FeedRpc.EmitRecommendTrack was not called")
	}

	got := feedRPC.trackReq
	if got.GetUserId() != 1001 ||
		got.GetEventType() != eventType ||
		got.GetContentId() != contentID ||
		got.GetRequestId() != requestID ||
		got.GetSnapshotId() != snapshotID ||
		got.GetVariantId() != variantID ||
		got.GetSource() != source ||
		got.GetPosition() != int32(position) ||
		got.GetFinalScore() != finalScore ||
		got.GetDwellMs() != dwellMs ||
		got.GetOccurredAt() != occurredAt {
		t.Fatalf("forwarded request = %+v", got)
	}
}

func TestEmitRecommendTrackRejectsBlankEvent(t *testing.T) {
	contentID := int64(2001)
	tests := []struct {
		name      string
		eventType *string
	}{
		{
			name:      "empty",
			eventType: ptrString(""),
		},
		{
			name:      "spaces",
			eventType: ptrString("   "),
		},
		{
			name:      "nil",
			eventType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feedRPC := &fakeFeedRPC{}
			logic := NewEmitRecommendTrackLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{FeedRpc: feedRPC},
			)

			resp, err := logic.EmitRecommendTrack(&types.RecommendTrackReq{
				EventType: tt.eventType,
				ContentId: &contentID,
			})
			if err == nil {
				t.Fatal("EmitRecommendTrack returned nil error")
			}
			if resp != nil {
				t.Fatalf("EmitRecommendTrack response = %+v, want nil", resp)
			}
			if feedRPC.trackReq != nil {
				t.Fatalf("FeedRpc.EmitRecommendTrack was called with %+v", feedRPC.trackReq)
			}
		})
	}
}

func TestEmitRecommendTrackRejectsInvalidContent(t *testing.T) {
	eventType := "click"
	tests := []struct {
		name      string
		contentID *int64
	}{
		{
			name:      "nil",
			contentID: nil,
		},
		{
			name:      "zero",
			contentID: ptrInt64(0),
		},
		{
			name:      "negative",
			contentID: ptrInt64(-1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feedRPC := &fakeFeedRPC{}
			logic := NewEmitRecommendTrackLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{FeedRpc: feedRPC},
			)

			resp, err := logic.EmitRecommendTrack(&types.RecommendTrackReq{
				EventType: &eventType,
				ContentId: tt.contentID,
			})
			if err == nil {
				t.Fatal("EmitRecommendTrack returned nil error")
			}
			if resp != nil {
				t.Fatalf("EmitRecommendTrack response = %+v, want nil", resp)
			}
			if feedRPC.trackReq != nil {
				t.Fatalf("FeedRpc.EmitRecommendTrack was called with %+v", feedRPC.trackReq)
			}
		})
	}
}

func TestEmitRecommendTrackRPCError(t *testing.T) {
	eventType := "click"
	contentID := int64(2001)
	rpcErr := errors.New("track rpc down")
	feedRPC := &fakeFeedRPC{trackErr: rpcErr}
	logic := NewEmitRecommendTrackLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.EmitRecommendTrack(&types.RecommendTrackReq{
		EventType: &eventType,
		ContentId: &contentID,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("EmitRecommendTrack error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("EmitRecommendTrack response = %+v, want nil", resp)
	}
	if feedRPC.trackReq == nil {
		t.Fatal("FeedRpc.EmitRecommendTrack was not called")
	}
}

func ptrString(value string) *string {
	return &value
}

func ptrInt64(value int64) *int64 {
	return &value
}
