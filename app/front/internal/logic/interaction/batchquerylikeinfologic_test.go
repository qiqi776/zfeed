package interaction

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"

	"google.golang.org/grpc"
)

type batchQueryLikeInfoService struct {
	fakeLikeService
	res *interactionpb.BatchLikeInfoRes
	err error
}

func (f *batchQueryLikeInfoService) BatchLikeInfo(
	_ context.Context,
	in *interactionpb.BatchLikeInfoReq,
	_ ...grpc.CallOption,
) (*interactionpb.BatchLikeInfoRes, error) {
	f.batchLikeReq = in
	return f.res, f.err
}

func TestBatchQueryLikeInfoRejectsInvalidContent(t *testing.T) {
	scene := "ARTICLE"
	validContentID := int64(2001)

	tests := []struct {
		name      string
		likeInfos []types.QueryLikeInfoReq
	}{
		{
			name: "zero",
			likeInfos: []types.QueryLikeInfoReq{
				{ContentId: int64Ptr(0), Scene: &scene},
			},
		},
		{
			name: "negative",
			likeInfos: []types.QueryLikeInfoReq{
				{ContentId: &validContentID, Scene: &scene},
				{ContentId: int64Ptr(-7), Scene: &scene},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			likeRPC := &fakeLikeService{}
			logic := NewBatchQueryLikeInfoLogic(
				context.Background(),
				&svc.ServiceContext{LikeRpc: likeRPC},
			)

			resp, err := logic.BatchQueryLikeInfo(&types.BatchQueryLikeInfoReq{
				LikeInfos: tt.likeInfos,
			})
			if err == nil {
				t.Fatal("BatchQueryLikeInfo returned nil error")
			}
			if resp != nil {
				t.Fatalf("BatchQueryLikeInfo response = %+v, want nil", resp)
			}
			if likeRPC.batchLikeReq != nil {
				t.Fatalf("LikeRpc.BatchLikeInfo was called with %+v", likeRPC.batchLikeReq)
			}
		})
	}
}

func TestBatchQueryLikeInfoMaps(t *testing.T) {
	articleID := int64(2001)
	commentID := int64(3002)
	articleScene := "ARTICLE"
	commentScene := "COMMENT"
	likeRPC := &batchQueryLikeInfoService{
		res: &interactionpb.BatchLikeInfoRes{
			LikeInfos: []*interactionpb.QueryLikeInfoRes{
				nil,
				{
					LikeCount: 7,
					IsLiked:   true,
					ContentId: articleID,
					Scene:     interactionpb.Scene_ARTICLE,
				},
				{
					LikeCount: 0,
					IsLiked:   false,
					ContentId: commentID,
					Scene:     interactionpb.Scene_COMMENT,
				},
			},
		},
	}
	logic := NewBatchQueryLikeInfoLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{LikeRpc: likeRPC},
	)

	resp, err := logic.BatchQueryLikeInfo(&types.BatchQueryLikeInfoReq{
		LikeInfos: []types.QueryLikeInfoReq{
			{ContentId: &articleID, Scene: &articleScene},
			{ContentId: &commentID, Scene: &commentScene},
		},
	})
	if err != nil {
		t.Fatalf("BatchQueryLikeInfo returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("BatchQueryLikeInfo returned nil response")
	}
	if likeRPC.batchLikeReq == nil {
		t.Fatal("LikeRpc.BatchLikeInfo was not called")
	}
	if likeRPC.batchLikeReq.GetUserId() != 1001 {
		t.Fatalf("rpc user_id = %d, want 1001", likeRPC.batchLikeReq.GetUserId())
	}
	if len(likeRPC.batchLikeReq.GetLikeInfos()) != 2 {
		t.Fatalf("len(rpc like_infos) = %d, want 2", len(likeRPC.batchLikeReq.GetLikeInfos()))
	}
	firstReq := likeRPC.batchLikeReq.GetLikeInfos()[0]
	secondReq := likeRPC.batchLikeReq.GetLikeInfos()[1]
	if firstReq.GetContentId() != articleID || firstReq.GetScene() != interactionpb.Scene_ARTICLE {
		t.Fatalf("first rpc like_info = %+v", firstReq)
	}
	if secondReq.GetContentId() != commentID || secondReq.GetScene() != interactionpb.Scene_COMMENT {
		t.Fatalf("second rpc like_info = %+v", secondReq)
	}
	if len(resp.LikeInfos) != 2 {
		t.Fatalf("len(response like_infos) = %d, want 2", len(resp.LikeInfos))
	}
	firstResp := resp.LikeInfos[0]
	secondResp := resp.LikeInfos[1]
	if firstResp.LikeCount != 7 || !firstResp.IsLiked || firstResp.ContentId != articleID || firstResp.Scene != "ARTICLE" {
		t.Fatalf("first response like_info = %+v", firstResp)
	}
	if secondResp.LikeCount != 0 || secondResp.IsLiked || secondResp.ContentId != commentID || secondResp.Scene != "COMMENT" {
		t.Fatalf("second response like_info = %+v", secondResp)
	}
}

func TestBatchQueryLikeInfoNilRPC(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	likeRPC := &batchQueryLikeInfoService{}
	logic := NewBatchQueryLikeInfoLogic(
		context.Background(),
		&svc.ServiceContext{LikeRpc: likeRPC},
	)

	resp, err := logic.BatchQueryLikeInfo(&types.BatchQueryLikeInfoReq{
		LikeInfos: []types.QueryLikeInfoReq{
			{ContentId: &contentID, Scene: &scene},
		},
	})
	if err == nil {
		t.Fatal("BatchQueryLikeInfo returned nil error")
	}
	if resp != nil {
		t.Fatalf("BatchQueryLikeInfo response = %+v, want nil", resp)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
