package content

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
	contentservice "zfeed/app/rpc/content/contentservice"
)

var _ contentservice.ContentService = (*fakeContentService)(nil)

type fakeContentService struct {
	publishArticleFunc func(ctx context.Context, in *contentpb.ArticlePublishReq, opts ...grpc.CallOption) (*contentpb.ArticlePublishRes, error)
	publishVideoFunc   func(ctx context.Context, in *contentpb.VideoPublishReq, opts ...grpc.CallOption) (*contentpb.VideoPublishRes, error)
	backfillInboxFunc  func(ctx context.Context, in *contentpb.BackfillFollowInboxReq, opts ...grpc.CallOption) (*contentpb.BackfillFollowInboxRes, error)
	uploadCredsFunc    func(ctx context.Context, in *contentpb.GetUploadCredentialsReq, opts ...grpc.CallOption) (*contentpb.GetUploadCredentialsRes, error)
	getDetailFunc      func(ctx context.Context, in *contentpb.GetContentDetailReq, opts ...grpc.CallOption) (*contentpb.GetContentDetailRes, error)
	editArticleFunc    func(ctx context.Context, in *contentpb.EditArticleReq, opts ...grpc.CallOption) (*contentpb.EditArticleRes, error)
	editVideoFunc      func(ctx context.Context, in *contentpb.EditVideoReq, opts ...grpc.CallOption) (*contentpb.EditVideoRes, error)
	deleteContentFunc  func(ctx context.Context, in *contentpb.DeleteContentReq, opts ...grpc.CallOption) (*contentpb.DeleteContentRes, error)
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

func (f *fakeContentService) BackfillFollowInbox(ctx context.Context, in *contentpb.BackfillFollowInboxReq, opts ...grpc.CallOption) (*contentpb.BackfillFollowInboxRes, error) {
	if f.backfillInboxFunc == nil {
		return nil, errors.New("unexpected BackfillFollowInbox call")
	}
	return f.backfillInboxFunc(ctx, in, opts...)
}

func (f *fakeContentService) GetUploadCredentials(ctx context.Context, in *contentpb.GetUploadCredentialsReq, opts ...grpc.CallOption) (*contentpb.GetUploadCredentialsRes, error) {
	if f.uploadCredsFunc == nil {
		return nil, errors.New("unexpected GetUploadCredentials call")
	}
	return f.uploadCredsFunc(ctx, in, opts...)
}

func (f *fakeContentService) GetContentDetail(ctx context.Context, in *contentpb.GetContentDetailReq, opts ...grpc.CallOption) (*contentpb.GetContentDetailRes, error) {
	if f.getDetailFunc == nil {
		return nil, errors.New("unexpected GetContentDetail call")
	}
	return f.getDetailFunc(ctx, in, opts...)
}

func (f *fakeContentService) EditArticle(ctx context.Context, in *contentpb.EditArticleReq, opts ...grpc.CallOption) (*contentpb.EditArticleRes, error) {
	if f.editArticleFunc == nil {
		return nil, errors.New("unexpected EditArticle call")
	}
	return f.editArticleFunc(ctx, in, opts...)
}

func (f *fakeContentService) EditVideo(ctx context.Context, in *contentpb.EditVideoReq, opts ...grpc.CallOption) (*contentpb.EditVideoRes, error) {
	if f.editVideoFunc == nil {
		return nil, errors.New("unexpected EditVideo call")
	}
	return f.editVideoFunc(ctx, in, opts...)
}

func (f *fakeContentService) DeleteContent(ctx context.Context, in *contentpb.DeleteContentReq, opts ...grpc.CallOption) (*contentpb.DeleteContentRes, error) {
	if f.deleteContentFunc == nil {
		return nil, errors.New("unexpected DeleteContent call")
	}
	return f.deleteContentFunc(ctx, in, opts...)
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

func TestPublishArticleLogic_AllowsMissingCover(t *testing.T) {
	const userID int64 = 1003

	svcCtx := &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			publishArticleFunc: func(ctx context.Context, in *contentpb.ArticlePublishReq, _ ...grpc.CallOption) (*contentpb.ArticlePublishRes, error) {
				if in.GetUserId() != userID {
					t.Fatalf("user_id = %d, want %d", in.GetUserId(), userID)
				}
				if in.GetCover() != "" {
					t.Fatalf("cover = %q, want empty string", in.GetCover())
				}
				return &contentpb.ArticlePublishRes{ContentId: 89}, nil
			},
		},
	}

	ctx := context.WithValue(context.Background(), "user_id", userID)
	logic := NewPublishArticleLogic(ctx, svcCtx)

	resp, err := logic.PublishArticle(&types.PublishArticleReq{
		Title:      strPtr("article without cover"),
		Content:    strPtr("hello article"),
		Visibility: int32Ptr(10),
	})
	if err != nil {
		t.Fatalf("PublishArticle returned error: %v", err)
	}
	if resp.ContentId != 89 {
		t.Fatalf("content_id = %d, want %d", resp.ContentId, 89)
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
