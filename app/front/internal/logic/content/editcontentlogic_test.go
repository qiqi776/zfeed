package content

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
)

func TestEditArticleCallsContentRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(1))
	logic := NewEditArticleLogic(ctx, &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			editArticleFunc: func(_ context.Context, in *contentpb.EditArticleReq, _ ...grpc.CallOption) (*contentpb.EditArticleRes, error) {
				if in.GetUserId() != 1 || in.GetContentId() != 101 {
					t.Fatalf("unexpected request: %+v", in)
				}
				if in.GetTitle() != "new-title" || in.GetContent() != "new body" {
					t.Fatalf("unexpected edit payload: %+v", in)
				}
				return &contentpb.EditArticleRes{ContentId: 101}, nil
			},
		},
	})

	resp, err := logic.EditArticle(&types.EditArticleReq{
		ContentId: 101,
		Title:     stringPtr("new-title"),
		Content:   stringPtr("new body"),
	})
	if err != nil {
		t.Fatalf("EditArticle returned error: %v", err)
	}
	if resp.ContentId != 101 {
		t.Fatalf("content_id = %d, want 101", resp.ContentId)
	}
}

func TestEditVideoCallsContentRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(2))
	logic := NewEditVideoLogic(ctx, &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			editVideoFunc: func(_ context.Context, in *contentpb.EditVideoReq, _ ...grpc.CallOption) (*contentpb.EditVideoRes, error) {
				if in.GetUserId() != 2 || in.GetContentId() != 202 {
					t.Fatalf("unexpected request: %+v", in)
				}
				if in.GetTitle() != "new-video" || in.GetOriginUrl() != "https://example.com/new.mp4" || in.GetDuration() != 66 {
					t.Fatalf("unexpected edit payload: %+v", in)
				}
				return &contentpb.EditVideoRes{ContentId: 202}, nil
			},
		},
	})

	resp, err := logic.EditVideo(&types.EditVideoReq{
		ContentId: 202,
		Title:     stringPtr("new-video"),
		VideoUrl:  stringPtr("https://example.com/new.mp4"),
		Duration:  editInt32Ptr(66),
	})
	if err != nil {
		t.Fatalf("EditVideo returned error: %v", err)
	}
	if resp.ContentId != 202 {
		t.Fatalf("content_id = %d, want 202", resp.ContentId)
	}
}

func stringPtr(value string) *string {
	return &value
}

func editInt32Ptr(value int32) *int32 {
	return &value
}
