package interaction

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"
)

type followUserService struct {
	fakeFollowService
	res *interactionpb.FollowUserRes
	err error
}

func (f *followUserService) FollowUser(
	_ context.Context,
	in *interactionpb.FollowUserReq,
	_ ...grpc.CallOption,
) (*interactionpb.FollowUserRes, error) {
	f.followReq = in
	return f.res, f.err
}

type fakeFollowService struct {
	followReq   *interactionpb.FollowUserReq
	unfollowReq *interactionpb.UnfollowUserReq
}

func (f *fakeFollowService) FollowUser(
	_ context.Context,
	in *interactionpb.FollowUserReq,
	_ ...grpc.CallOption,
) (*interactionpb.FollowUserRes, error) {
	f.followReq = in
	return &interactionpb.FollowUserRes{IsFollowed: true}, nil
}

func (f *fakeFollowService) UnfollowUser(
	_ context.Context,
	in *interactionpb.UnfollowUserReq,
	_ ...grpc.CallOption,
) (*interactionpb.UnfollowUserRes, error) {
	f.unfollowReq = in
	return &interactionpb.UnfollowUserRes{IsFollowed: false}, nil
}

func (f *fakeFollowService) ListFollowees(
	context.Context,
	*interactionpb.ListFolloweesReq,
	...grpc.CallOption,
) (*interactionpb.ListFolloweesRes, error) {
	return &interactionpb.ListFolloweesRes{}, nil
}

func (f *fakeFollowService) ListFollowers(
	context.Context,
	*interactionpb.ListFollowersReq,
	...grpc.CallOption,
) (*interactionpb.ListFollowersRes, error) {
	return &interactionpb.ListFollowersRes{}, nil
}

func (f *fakeFollowService) BatchQueryFollowing(
	context.Context,
	*interactionpb.BatchQueryFollowingReq,
	...grpc.CallOption,
) (*interactionpb.BatchQueryFollowingRes, error) {
	return &interactionpb.BatchQueryFollowingRes{}, nil
}

func (f *fakeFollowService) GetFollowSummary(
	context.Context,
	*interactionpb.GetFollowSummaryReq,
	...grpc.CallOption,
) (*interactionpb.GetFollowSummaryRes, error) {
	return &interactionpb.GetFollowSummaryRes{}, nil
}

func TestFollowUserRejectsInvalidTarget(t *testing.T) {
	tests := []struct {
		name         string
		targetUserID int64
	}{
		{name: "zero", targetUserID: 0},
		{name: "negative", targetUserID: -7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followRPC := &fakeFollowService{}
			logic := NewFollowUserLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{FollowRpc: followRPC},
			)

			resp, err := logic.FollowUser(&types.FollowUserReq{
				TargetUserId: &tt.targetUserID,
			})
			if err == nil {
				t.Fatal("FollowUser returned nil error")
			}
			if resp != nil {
				t.Fatalf("FollowUser response = %+v, want nil", resp)
			}
			if followRPC.followReq != nil {
				t.Fatalf("FollowRpc.FollowUser was called with %+v", followRPC.followReq)
			}
		})
	}
}

func TestFollowUserRejectsSelf(t *testing.T) {
	targetUserID := int64(1001)
	followRPC := &fakeFollowService{}
	logic := NewFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.FollowUser(&types.FollowUserReq{
		TargetUserId: &targetUserID,
	})
	if err == nil {
		t.Fatal("FollowUser returned nil error")
	}
	if resp != nil {
		t.Fatalf("FollowUser response = %+v, want nil", resp)
	}
	if followRPC.followReq != nil {
		t.Fatalf("FollowRpc.FollowUser was called with %+v", followRPC.followReq)
	}
}

func TestFollowUserMaps(t *testing.T) {
	targetUserID := int64(2001)
	followRPC := &fakeFollowService{}
	logic := NewFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.FollowUser(&types.FollowUserReq{
		TargetUserId: &targetUserID,
	})
	if err != nil {
		t.Fatalf("FollowUser returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("FollowUser returned nil response")
	}
	if !resp.IsFollowed {
		t.Fatalf("FollowUser IsFollowed = %v, want true", resp.IsFollowed)
	}
	if followRPC.followReq == nil {
		t.Fatal("FollowRpc.FollowUser was not called")
	}
	if followRPC.followReq.GetUserId() != 1001 || followRPC.followReq.GetFollowUserId() != targetUserID {
		t.Fatalf("rpc request = %+v", followRPC.followReq)
	}
}

func TestFollowUserRPCError(t *testing.T) {
	targetUserID := int64(2001)
	rpcErr := errors.New("follow rpc down")
	followRPC := &followUserService{err: rpcErr}
	logic := NewFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.FollowUser(&types.FollowUserReq{
		TargetUserId: &targetUserID,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("FollowUser error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("FollowUser response = %+v, want nil", resp)
	}
}

func TestFollowUserNilRPC(t *testing.T) {
	targetUserID := int64(2001)
	followRPC := &followUserService{}
	logic := NewFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.FollowUser(&types.FollowUserReq{
		TargetUserId: &targetUserID,
	})
	if err == nil {
		t.Fatal("FollowUser returned nil error")
	}
	if resp != nil {
		t.Fatalf("FollowUser response = %+v, want nil", resp)
	}
}
