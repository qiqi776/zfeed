package interaction

import (
	"context"
	"errors"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"

	"google.golang.org/grpc"
)

type unfollowUserService struct {
	fakeFollowService
	res *interactionpb.UnfollowUserRes
	err error
}

func (f *unfollowUserService) UnfollowUser(
	_ context.Context,
	in *interactionpb.UnfollowUserReq,
	_ ...grpc.CallOption,
) (*interactionpb.UnfollowUserRes, error) {
	f.unfollowReq = in
	return f.res, f.err
}

func TestUnFollowUserRejectsInvalidTarget(t *testing.T) {
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
			logic := NewUnFollowUserLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{FollowRpc: followRPC},
			)

			resp, err := logic.UnFollowUser(&types.UnFollowUserReq{
				TargetUserId: &tt.targetUserID,
			})
			if err == nil {
				t.Fatal("UnFollowUser returned nil error")
			}
			if resp != nil {
				t.Fatalf("UnFollowUser response = %+v, want nil", resp)
			}
			if followRPC.unfollowReq != nil {
				t.Fatalf("FollowRpc.UnfollowUser was called with %+v", followRPC.unfollowReq)
			}
		})
	}
}

func TestUnFollowUserRejectsSelf(t *testing.T) {
	targetUserID := int64(1001)
	followRPC := &fakeFollowService{}
	logic := NewUnFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.UnFollowUser(&types.UnFollowUserReq{
		TargetUserId: &targetUserID,
	})
	if err == nil {
		t.Fatal("UnFollowUser returned nil error")
	}
	if resp != nil {
		t.Fatalf("UnFollowUser response = %+v, want nil", resp)
	}
	if followRPC.unfollowReq != nil {
		t.Fatalf("FollowRpc.UnfollowUser was called with %+v", followRPC.unfollowReq)
	}
}

func TestUnFollowUserMaps(t *testing.T) {
	targetUserID := int64(2001)
	followRPC := &fakeFollowService{}
	logic := NewUnFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.UnFollowUser(&types.UnFollowUserReq{
		TargetUserId: &targetUserID,
	})
	if err != nil {
		t.Fatalf("UnFollowUser returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("UnFollowUser returned nil response")
	}
	if resp.IsFollowed {
		t.Fatalf("UnFollowUser IsFollowed = %v, want false", resp.IsFollowed)
	}
	if followRPC.unfollowReq == nil {
		t.Fatal("FollowRpc.UnfollowUser was not called")
	}
	if followRPC.unfollowReq.GetUserId() != 1001 || followRPC.unfollowReq.GetFollowUserId() != targetUserID {
		t.Fatalf("rpc request = %+v", followRPC.unfollowReq)
	}
}

func TestUnFollowUserRPCError(t *testing.T) {
	targetUserID := int64(2001)
	rpcErr := errors.New("follow rpc down")
	followRPC := &unfollowUserService{err: rpcErr}
	logic := NewUnFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.UnFollowUser(&types.UnFollowUserReq{
		TargetUserId: &targetUserID,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("UnFollowUser error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("UnFollowUser response = %+v, want nil", resp)
	}
}

func TestUnFollowUserNilRPC(t *testing.T) {
	targetUserID := int64(2001)
	followRPC := &unfollowUserService{}
	logic := NewUnFollowUserLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FollowRpc: followRPC},
	)

	resp, err := logic.UnFollowUser(&types.UnFollowUserReq{
		TargetUserId: &targetUserID,
	})
	if err == nil {
		t.Fatal("UnFollowUser returned nil error")
	}
	if resp != nil {
		t.Fatalf("UnFollowUser response = %+v, want nil", resp)
	}
}
