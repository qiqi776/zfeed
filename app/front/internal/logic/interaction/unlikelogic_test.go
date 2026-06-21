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

type unlikeActionService struct {
	fakeLikeService
	res *interactionpb.UnlikeRes
	err error
}

func (f *unlikeActionService) Unlike(
	_ context.Context,
	in *interactionpb.UnlikeReq,
	_ ...grpc.CallOption,
) (*interactionpb.UnlikeRes, error) {
	f.unlikeReq = in
	return f.res, f.err
}

func TestUnlikeRejectsInvalidContent(t *testing.T) {
	scene := "ARTICLE"
	tests := []struct {
		name      string
		contentID int64
	}{
		{name: "zero", contentID: 0},
		{name: "negative", contentID: -7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			likeRPC := &fakeLikeService{}
			logic := NewUnlikeLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{LikeRpc: likeRPC},
			)

			resp, err := logic.Unlike(&types.UnlikeReq{
				ContentId: &tt.contentID,
				Scene:     &scene,
			})
			if err == nil {
				t.Fatal("Unlike returned nil error")
			}
			if resp != nil {
				t.Fatalf("Unlike response = %+v, want nil", resp)
			}
			if likeRPC.unlikeReq != nil {
				t.Fatalf("LikeRpc.Unlike was called with %+v", likeRPC.unlikeReq)
			}
		})
	}
}

func TestUnlikeMaps(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	likeRPC := &fakeLikeService{}
	logic := NewUnlikeLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{LikeRpc: likeRPC},
	)

	resp, err := logic.Unlike(&types.UnlikeReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("Unlike returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Unlike returned nil response")
	}
	if likeRPC.unlikeReq == nil {
		t.Fatal("LikeRpc.Unlike was not called")
	}
	if likeRPC.unlikeReq.GetUserId() != 1001 || likeRPC.unlikeReq.GetContentId() != contentID ||
		likeRPC.unlikeReq.GetScene() != interactionpb.Scene_ARTICLE {
		t.Fatalf("rpc request = %+v", likeRPC.unlikeReq)
	}
}

func TestUnlikeRPCError(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	rpcErr := errors.New("like rpc down")
	likeRPC := &unlikeActionService{err: rpcErr}
	logic := NewUnlikeLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{LikeRpc: likeRPC},
	)

	resp, err := logic.Unlike(&types.UnlikeReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("Unlike error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("Unlike response = %+v, want nil", resp)
	}
}

func TestUnlikeNilRPC(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	likeRPC := &unlikeActionService{}
	logic := NewUnlikeLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{LikeRpc: likeRPC},
	)

	resp, err := logic.Unlike(&types.UnlikeReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err == nil {
		t.Fatal("Unlike returned nil error")
	}
	if resp != nil {
		t.Fatalf("Unlike response = %+v, want nil", resp)
	}
}
