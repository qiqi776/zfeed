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

type removeFavoriteActionService struct {
	fakeFavoriteService
	res *interactionpb.RemoveFavoriteRes
	err error
}

func (f *removeFavoriteActionService) RemoveFavorite(
	_ context.Context,
	in *interactionpb.RemoveFavoriteReq,
	_ ...grpc.CallOption,
) (*interactionpb.RemoveFavoriteRes, error) {
	f.removeFavoriteReq = in
	return f.res, f.err
}

func TestRemoveFavoriteRejectsInvalidContent(t *testing.T) {
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
			favoriteRPC := &fakeFavoriteService{}
			logic := NewRemoveFavoriteLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{FavoriteRpc: favoriteRPC},
			)

			resp, err := logic.RemoveFavorite(&types.RemoveFavoriteReq{
				ContentId: &tt.contentID,
				Scene:     &scene,
			})
			if err == nil {
				t.Fatal("RemoveFavorite returned nil error")
			}
			if resp != nil {
				t.Fatalf("RemoveFavorite response = %+v, want nil", resp)
			}
			if favoriteRPC.removeFavoriteReq != nil {
				t.Fatalf("FavoriteRpc.RemoveFavorite was called with %+v", favoriteRPC.removeFavoriteReq)
			}
		})
	}
}

func TestRemoveFavoriteMaps(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	favoriteRPC := &fakeFavoriteService{}
	logic := NewRemoveFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.RemoveFavorite(&types.RemoveFavoriteReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("RemoveFavorite returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("RemoveFavorite returned nil response")
	}
	if favoriteRPC.removeFavoriteReq == nil {
		t.Fatal("FavoriteRpc.RemoveFavorite was not called")
	}
	if favoriteRPC.removeFavoriteReq.GetUserId() != 1001 || favoriteRPC.removeFavoriteReq.GetContentId() != contentID ||
		favoriteRPC.removeFavoriteReq.GetScene() != interactionpb.Scene_ARTICLE {
		t.Fatalf("rpc request = %+v", favoriteRPC.removeFavoriteReq)
	}
}

func TestRemoveFavoriteRPCError(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	rpcErr := errors.New("favorite rpc down")
	favoriteRPC := &removeFavoriteActionService{err: rpcErr}
	logic := NewRemoveFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.RemoveFavorite(&types.RemoveFavoriteReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("RemoveFavorite error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("RemoveFavorite response = %+v, want nil", resp)
	}
}

func TestRemoveFavoriteNilRPC(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	favoriteRPC := &removeFavoriteActionService{}
	logic := NewRemoveFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.RemoveFavorite(&types.RemoveFavoriteReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err == nil {
		t.Fatal("RemoveFavorite returned nil error")
	}
	if resp != nil {
		t.Fatalf("RemoveFavorite response = %+v, want nil", resp)
	}
}
