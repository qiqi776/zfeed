package feed

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
)

type fakeFeedRPC struct {
	trackReq *contentpb.EmitRecommendTrackReq
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
	return &contentpb.EmitRecommendTrackRes{}, nil
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

func TestEmitRecommendTrackForwardsOptionalUserAndFields(t *testing.T) {
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
