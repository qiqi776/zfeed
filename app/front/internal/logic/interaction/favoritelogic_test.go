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

type favoriteActionService struct {
	fakeFavoriteService
	res *interactionpb.FavoriteRes
	err error
}

func (f *favoriteActionService) Favorite(
	_ context.Context,
	in *interactionpb.FavoriteReq,
	_ ...grpc.CallOption,
) (*interactionpb.FavoriteRes, error) {
	f.favoriteReq = in
	return f.res, f.err
}

func TestFavoriteMaps(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	favoriteRPC := &fakeFavoriteService{}
	logic := NewFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.Favorite(&types.FavoriteReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("Favorite returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Favorite returned nil response")
	}
	if favoriteRPC.favoriteReq == nil {
		t.Fatal("FavoriteRpc.Favorite was not called")
	}
	if favoriteRPC.favoriteReq.GetUserId() != 1001 || favoriteRPC.favoriteReq.GetContentId() != contentID ||
		favoriteRPC.favoriteReq.GetScene() != interactionpb.Scene_ARTICLE {
		t.Fatalf("rpc request = %+v", favoriteRPC.favoriteReq)
	}
}

func TestFavoriteRPCError(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	rpcErr := errors.New("favorite rpc down")
	favoriteRPC := &favoriteActionService{err: rpcErr}
	logic := NewFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.Favorite(&types.FavoriteReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("Favorite error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("Favorite response = %+v, want nil", resp)
	}
}

func TestFavoriteNilRPC(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	favoriteRPC := &favoriteActionService{}
	logic := NewFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.Favorite(&types.FavoriteReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err == nil {
		t.Fatal("Favorite returned nil error")
	}
	if resp != nil {
		t.Fatalf("Favorite response = %+v, want nil", resp)
	}
}
