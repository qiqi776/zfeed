package content

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
)

func TestGetContentDetailPassesViewerIDToRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(1001))
	logic := NewGetContentDetailLogic(ctx, &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			getDetailFunc: func(_ context.Context, in *contentpb.GetContentDetailReq, _ ...grpc.CallOption) (*contentpb.GetContentDetailRes, error) {
				if in.GetContentId() != 5001 {
					t.Fatalf("content_id = %d, want 5001", in.GetContentId())
				}
				if in.GetViewerId() != 1001 {
					t.Fatalf("viewer_id = %d, want 1001", in.GetViewerId())
				}
				return &contentpb.GetContentDetailRes{
					Detail: &contentpb.ContentDetail{
						ContentId:         5001,
						ContentType:       contentpb.ContentType_CONTENT_TYPE_ARTICLE,
						AuthorId:          1001,
						AuthorName:        "author",
						AuthorAvatar:      "avatar",
						Title:             "private article",
						CoverUrl:          "cover",
						ArticleContent:    "body",
						IsFollowingAuthor: true,
					},
				}, nil
			},
		},
	})

	resp, err := logic.GetContentDetail(&types.GetContentDetailReq{ContentId: int64Ptr(5001)})
	if err != nil {
		t.Fatalf("GetContentDetail returned error: %v", err)
	}
	if resp.Detail.ContentId != 5001 || resp.Detail.Title != "private article" {
		t.Fatalf("unexpected detail: %+v", resp.Detail)
	}
	if !resp.Detail.IsFollowingAuthor {
		t.Fatal("expected author follow state to be mapped")
	}
}

func TestGetContentDetailRejectsInvalidRequest(t *testing.T) {
	logic := NewGetContentDetailLogic(context.Background(), &svc.ServiceContext{})
	if _, err := logic.GetContentDetail(&types.GetContentDetailReq{}); err == nil {
		t.Fatal("expected invalid request error")
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
