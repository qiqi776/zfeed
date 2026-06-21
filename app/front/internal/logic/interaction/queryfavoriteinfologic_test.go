package interaction

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"

	"google.golang.org/grpc"
)

type queryFavoriteInfoService struct {
	fakeFavoriteService
	res *interactionpb.QueryFavoriteInfoRes
	err error
}

func (f *queryFavoriteInfoService) QueryFavoriteInfo(
	_ context.Context,
	in *interactionpb.QueryFavoriteInfoReq,
	_ ...grpc.CallOption,
) (*interactionpb.QueryFavoriteInfoRes, error) {
	f.queryFavoriteReq = in
	return f.res, f.err
}

func TestQueryFavoriteInfoRejectsInvalidContent(t *testing.T) {
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
			logic := NewQueryFavoriteInfoLogic(
				context.Background(),
				&svc.ServiceContext{FavoriteRpc: favoriteRPC},
			)

			resp, err := logic.QueryFavoriteInfo(&types.QueryFavoriteInfoReq{
				ContentId: &tt.contentID,
				Scene:     &scene,
			})
			if err == nil {
				t.Fatal("QueryFavoriteInfo returned nil error")
			}
			if resp != nil {
				t.Fatalf("QueryFavoriteInfo response = %+v, want nil", resp)
			}
			if favoriteRPC.queryFavoriteReq != nil {
				t.Fatalf("FavoriteRpc.QueryFavoriteInfo was called with %+v", favoriteRPC.queryFavoriteReq)
			}
		})
	}
}

func TestQueryFavoriteInfoMaps(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	favoriteRPC := &queryFavoriteInfoService{
		res: &interactionpb.QueryFavoriteInfoRes{
			FavoriteCount: 11,
			IsFavorited:   true,
			ContentId:     contentID,
			Scene:         interactionpb.Scene_ARTICLE,
		},
	}
	logic := NewQueryFavoriteInfoLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.QueryFavoriteInfo(&types.QueryFavoriteInfoReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("QueryFavoriteInfo returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("QueryFavoriteInfo returned nil response")
	}
	if favoriteRPC.queryFavoriteReq == nil {
		t.Fatal("FavoriteRpc.QueryFavoriteInfo was not called")
	}
	if favoriteRPC.queryFavoriteReq.GetUserId() != 1001 || favoriteRPC.queryFavoriteReq.GetContentId() != contentID ||
		favoriteRPC.queryFavoriteReq.GetScene() != interactionpb.Scene_ARTICLE {
		t.Fatalf("rpc request = %+v", favoriteRPC.queryFavoriteReq)
	}
	if resp.FavoriteCount != 11 || !resp.IsFavorite || resp.ContentId != contentID || resp.Scene != "ARTICLE" {
		t.Fatalf("QueryFavoriteInfo response = %+v", resp)
	}
}

func TestQueryFavoriteInfoNilRPC(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	favoriteRPC := &queryFavoriteInfoService{}
	logic := NewQueryFavoriteInfoLogic(
		context.Background(),
		&svc.ServiceContext{FavoriteRpc: favoriteRPC},
	)

	resp, err := logic.QueryFavoriteInfo(&types.QueryFavoriteInfoReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err == nil {
		t.Fatal("QueryFavoriteInfo returned nil error")
	}
	if resp != nil {
		t.Fatalf("QueryFavoriteInfo response = %+v, want nil", resp)
	}
}
