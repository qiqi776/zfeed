package interaction

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"

	"google.golang.org/grpc"
)

type queryLikeInfoService struct {
	fakeLikeService
	res *interactionpb.QueryLikeInfoRes
	err error
}

func (f *queryLikeInfoService) QueryLikeInfo(
	_ context.Context,
	in *interactionpb.QueryLikeInfoReq,
	_ ...grpc.CallOption,
) (*interactionpb.QueryLikeInfoRes, error) {
	f.queryLikeReq = in
	return f.res, f.err
}

func TestQueryLikeInfoRejectsInvalidContent(t *testing.T) {
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
			logic := NewQueryLikeInfoLogic(
				context.Background(),
				&svc.ServiceContext{LikeRpc: likeRPC},
			)

			resp, err := logic.QueryLikeInfo(&types.QueryLikeInfoReq{
				ContentId: &tt.contentID,
				Scene:     &scene,
			})
			if err == nil {
				t.Fatal("QueryLikeInfo returned nil error")
			}
			if resp != nil {
				t.Fatalf("QueryLikeInfo response = %+v, want nil", resp)
			}
			if likeRPC.queryLikeReq != nil {
				t.Fatalf("LikeRpc.QueryLikeInfo was called with %+v", likeRPC.queryLikeReq)
			}
		})
	}
}

func TestQueryLikeInfoMaps(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	likeRPC := &queryLikeInfoService{
		res: &interactionpb.QueryLikeInfoRes{
			LikeCount: 9,
			IsLiked:   true,
			ContentId: contentID,
			Scene:     interactionpb.Scene_ARTICLE,
		},
	}
	logic := NewQueryLikeInfoLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{LikeRpc: likeRPC},
	)

	resp, err := logic.QueryLikeInfo(&types.QueryLikeInfoReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("QueryLikeInfo returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("QueryLikeInfo returned nil response")
	}
	if likeRPC.queryLikeReq == nil {
		t.Fatal("LikeRpc.QueryLikeInfo was not called")
	}
	if likeRPC.queryLikeReq.GetUserId() != 1001 || likeRPC.queryLikeReq.GetContentId() != contentID ||
		likeRPC.queryLikeReq.GetScene() != interactionpb.Scene_ARTICLE {
		t.Fatalf("rpc request = %+v", likeRPC.queryLikeReq)
	}
	if resp.LikeCount != 9 || !resp.IsLiked || resp.ContentId != contentID || resp.Scene != "ARTICLE" {
		t.Fatalf("QueryLikeInfo response = %+v", resp)
	}
}

func TestQueryLikeInfoNilRPC(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	likeRPC := &queryLikeInfoService{}
	logic := NewQueryLikeInfoLogic(
		context.Background(),
		&svc.ServiceContext{LikeRpc: likeRPC},
	)

	resp, err := logic.QueryLikeInfo(&types.QueryLikeInfoReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err == nil {
		t.Fatal("QueryLikeInfo returned nil error")
	}
	if resp != nil {
		t.Fatalf("QueryLikeInfo response = %+v, want nil", resp)
	}
}
