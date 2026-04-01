package content

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	contentpb "zfeed/app/rpc/content/content"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

var _ contentservice.ContentService = (*fakeContentService)(nil)

type fakeContentService struct {
	publishArticleFunc func(ctx context.Context, in *contentpb.ArticlePublishReq, opts ...grpc.CallOption) (*contentpb.ArticlePublishRes, error)
	publishVideoFunc   func(ctx context.Context, in *contentpb.VideoPublishReq, opts ...grpc.CallOption) (*contentpb.VideoPublishRes, error)
}

func (f *fakeContentService) PublishArticle(ctx context.Context, in *contentpb.ArticlePublishReq, opts ...grpc.CallOption) (*contentpb.ArticlePublishRes, error) {
	if f.publishArticleFunc == nil {
		return nil, errors.New("unexpected PublishArticle call")
	}
	return f.publishArticleFunc(ctx, in, opts...)
}

func (f *fakeContentService) PublishVideo(ctx context.Context, in *contentpb.VideoPublishReq, opts ...grpc.CallOption) (*contentpb.VideoPublishRes, error) {
	if f.publishVideoFunc == nil {
		return nil, errors.New("unexpected PublishVideo call")
	}
	return f.publishVideoFunc(ctx, in, opts...)
}

func strPtr(v string) *string {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
}

func TestPublishArticleLogic_PassesUserIDToRPC(t *testing.T) {
	const userID int64 = 1001
	called := false

	svcCtx := &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			publishArticleFunc: func(ctx context.Context, in *contentpb.ArticlePublishReq, _ ...grpc.CallOption) (*contentpb.ArticlePublishRes, error) {
				called = true
				if in.GetUserId() != userID {
					t.Fatalf("user_id = %d, want %d", in.GetUserId(), userID)
				}
				if in.GetTitle() != "article" {
					t.Fatalf("title = %q, want %q", in.GetTitle(), "article")
				}
				return &contentpb.ArticlePublishRes{ContentId: 88}, nil
			},
		},
	}

	ctx := context.WithValue(context.Background(), "user_id", userID)
	logic := NewPublishArticleLogic(ctx, svcCtx)

	resp, err := logic.PublishArticle(&types.PublishArticleReq{
		Title:      strPtr("article"),
		Cover:      strPtr("https://example.com/a.png"),
		Content:    strPtr("hello article"),
		Visibility: int32Ptr(10),
	})
	if err != nil {
		t.Fatalf("PublishArticle returned error: %v", err)
	}
	if !called {
		t.Fatal("content rpc was not called")
	}
	if resp.ContentId != 88 {
		t.Fatalf("content_id = %d, want %d", resp.ContentId, 88)
	}
}

func TestPublishVideoLogic_PassesUserIDToRPC(t *testing.T) {
	const userID int64 = 1002
	called := false
	duration := int32(120)

	svcCtx := &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			publishVideoFunc: func(ctx context.Context, in *contentpb.VideoPublishReq, _ ...grpc.CallOption) (*contentpb.VideoPublishRes, error) {
				called = true
				if in.GetUserId() != userID {
					t.Fatalf("user_id = %d, want %d", in.GetUserId(), userID)
				}
				if in.GetOriginUrl() != "https://example.com/v.mp4" {
					t.Fatalf("origin_url = %q, want %q", in.GetOriginUrl(), "https://example.com/v.mp4")
				}
				if in.GetDuration() != duration {
					t.Fatalf("duration = %d, want %d", in.GetDuration(), duration)
				}
				return &contentpb.VideoPublishRes{ContentId: 99}, nil
			},
		},
	}

	ctx := context.WithValue(context.Background(), "user_id", userID)
	logic := NewPublishVideoLogic(ctx, svcCtx)

	resp, err := logic.PublishVideo(&types.PublishVideoReq{
		Title:      strPtr("video"),
		VideoUrl:   strPtr("https://example.com/v.mp4"),
		CoverUrl:   strPtr("https://example.com/v.png"),
		Duration:   &duration,
		Visibility: int32Ptr(10),
	})
	if err != nil {
		t.Fatalf("PublishVideo returned error: %v", err)
	}
	if !called {
		t.Fatal("content rpc was not called")
	}
	if resp.ContentId != 99 {
		t.Fatalf("content_id = %d, want %d", resp.ContentId, 99)
	}
}
