package user

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	followservicepb "zfeed/app/rpc/interaction/client/followservice"
	interactionpb "zfeed/app/rpc/interaction/interaction"
)

type stubFollowService struct {
	listFollowersFunc func(ctx context.Context, in *followservicepb.ListFollowersReq, opts ...grpc.CallOption) (*followservicepb.ListFollowersRes, error)
}

func (s *stubFollowService) FollowUser(context.Context, *followservicepb.FollowUserReq, ...grpc.CallOption) (*followservicepb.FollowUserRes, error) {
	return &followservicepb.FollowUserRes{}, nil
}

func (s *stubFollowService) UnfollowUser(context.Context, *followservicepb.UnfollowUserReq, ...grpc.CallOption) (*followservicepb.UnfollowUserRes, error) {
	return &followservicepb.UnfollowUserRes{}, nil
}

func (s *stubFollowService) ListFollowees(context.Context, *followservicepb.ListFolloweesReq, ...grpc.CallOption) (*followservicepb.ListFolloweesRes, error) {
	return &followservicepb.ListFolloweesRes{}, nil
}

func (s *stubFollowService) ListFollowers(ctx context.Context, in *followservicepb.ListFollowersReq, opts ...grpc.CallOption) (*followservicepb.ListFollowersRes, error) {
	return s.listFollowersFunc(ctx, in, opts...)
}

func (s *stubFollowService) BatchQueryFollowing(context.Context, *followservicepb.BatchQueryFollowingReq, ...grpc.CallOption) (*followservicepb.BatchQueryFollowingRes, error) {
	return &followservicepb.BatchQueryFollowingRes{}, nil
}

func (s *stubFollowService) GetFollowSummary(context.Context, *followservicepb.GetFollowSummaryReq, ...grpc.CallOption) (*followservicepb.GetFollowSummaryRes, error) {
	return &followservicepb.GetFollowSummaryRes{}, nil
}

func TestQueryFollowersCallsFollowRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(3001))
	logic := NewQueryFollowersLogic(ctx, &svc.ServiceContext{
		FollowRpc: &stubFollowService{
			listFollowersFunc: func(_ context.Context, in *followservicepb.ListFollowersReq, _ ...grpc.CallOption) (*followservicepb.ListFollowersRes, error) {
				if in.GetUserId() != 2001 || in.GetViewerId() != 3001 || in.GetPageSize() != 2 {
					t.Fatalf("unexpected rpc request: %+v", in)
				}
				return &followservicepb.ListFollowersRes{
					Items: []*interactionpb.FollowerProfile{
						{UserId: 1003, Nickname: "u1003", Avatar: "a3", Bio: "b3", IsFollowing: false},
						{UserId: 1002, Nickname: "u1002", Avatar: "a2", Bio: "b2", IsFollowing: true},
					},
					NextCursor: 1002,
					HasMore:    true,
				}, nil
			},
		},
	})

	resp, err := logic.QueryFollowers(&types.QueryFollowersReq{
		UserId:   queryFollowersInt64Ptr(2001),
		PageSize: uint32Ptr(2),
	})
	if err != nil {
		t.Fatalf("QueryFollowers returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.Items))
	}
	if resp.Items[0].UserId != 1003 || resp.Items[1].UserId != 1002 || !resp.Items[1].IsFollowing {
		t.Fatalf("unexpected items: %+v", resp.Items)
	}
	if !resp.HasMore || resp.NextCursor != 1002 {
		t.Fatalf("unexpected pagination: %+v", resp)
	}
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}

func queryFollowersInt64Ptr(value int64) *int64 {
	return &value
}
