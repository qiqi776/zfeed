package contentdetail

import (
	"context"
	"errors"
	"net"
	"testing"

	contentpb "zfeed/app/rpc/content/content"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestContentDetailBenchServerReturnsFixture(t *testing.T) {
	t.Parallel()

	server := newContentDetailBenchServer()
	viewerID := int64(10001)
	resp, err := server.GetContentDetail(context.Background(), &contentpb.GetContentDetailReq{
		ContentId: 900001,
		ViewerId:  &viewerID,
	})
	if err != nil {
		t.Fatalf("GetContentDetail returned error: %v", err)
	}

	detail := resp.GetDetail()
	if detail.GetContentId() != 900001 {
		t.Fatalf("content id = %d, want 900001", detail.GetContentId())
	}
	if detail.GetTitle() == "" || detail.GetAuthorId() == 0 {
		t.Fatalf("detail missing fixture fields: %+v", detail)
	}
}

func BenchmarkContentDetailBuildFixture(b *testing.B) {
	server := newContentDetailBenchServer()
	req := &contentpb.GetContentDetailReq{
		ContentId: 900001,
		ViewerId:  int64Ptr(10001),
	}

	b.ReportAllocs()
	for b.Loop() {
		resp, err := server.GetContentDetail(context.Background(), req)
		if err != nil {
			b.Fatalf("GetContentDetail returned error: %v", err)
		}
		if resp.GetDetail().GetContentId() != req.GetContentId() {
			b.Fatalf("content id = %d, want %d", resp.GetDetail().GetContentId(), req.GetContentId())
		}
	}
}

func BenchmarkContentDetailGRPCUnary(b *testing.B) {
	ctx := context.Background()
	server := grpc.NewServer()
	contentpb.RegisterContentServiceServer(server, newContentDetailBenchServer())
	client, cleanup := newContentDetailBenchClient(b, server)
	defer cleanup()

	req := &contentpb.GetContentDetailReq{
		ContentId: 900001,
		ViewerId:  int64Ptr(10001),
	}

	b.ReportAllocs()
	for b.Loop() {
		resp, err := client.GetContentDetail(ctx, req)
		if err != nil {
			b.Fatalf("GetContentDetail returned error: %v", err)
		}
		if resp.GetDetail().GetContentId() != req.GetContentId() {
			b.Fatalf("content id = %d, want %d", resp.GetDetail().GetContentId(), req.GetContentId())
		}
	}
}

func newContentDetailBenchClient(b *testing.B, server *grpc.Server) (contentpb.ContentServiceClient, func()) {
	b.Helper()

	listener := bufconn.Listen(1024 * 1024)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	conn, err := grpc.DialContext(
		context.Background(),
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		server.Stop()
		_ = listener.Close()
		b.Fatalf("dial content detail bench server: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
		if err := <-serveDone; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			b.Fatalf("content detail bench server stopped with error: %v", err)
		}
	}

	return contentpb.NewContentServiceClient(conn), cleanup
}

type contentDetailBenchServer struct {
	contentpb.UnimplementedContentServiceServer
	detail *contentpb.ContentDetail
}

func newContentDetailBenchServer() *contentDetailBenchServer {
	return &contentDetailBenchServer{
		detail: &contentpb.ContentDetail{
			ContentId:         900001,
			ContentType:       contentpb.ContentType_ARTICLE,
			AuthorId:          10001,
			AuthorName:        "bench_user_10001",
			AuthorAvatar:      "https://example.com/bench/avatar-10001.png",
			Title:             "bench_article_900001",
			Description:       "bench content detail fixture",
			CoverUrl:          "https://example.com/bench/cover-900001.png",
			ArticleContent:    "bench article body for content detail benchmark",
			PublishedAt:       1781952000,
			LikeCount:         1234,
			FavoriteCount:     456,
			CommentCount:      789,
			IsLiked:           true,
			IsFavorited:       true,
			IsFollowingAuthor: true,
		},
	}
}

func (s *contentDetailBenchServer) GetContentDetail(context.Context, *contentpb.GetContentDetailReq) (*contentpb.GetContentDetailRes, error) {
	detail := *s.detail
	return &contentpb.GetContentDetailRes{Detail: &detail}, nil
}

func (s *contentDetailBenchServer) PublishArticle(context.Context, *contentpb.ArticlePublishReq) (*contentpb.ArticlePublishRes, error) {
	return nil, unimplemented()
}

func (s *contentDetailBenchServer) PublishVideo(context.Context, *contentpb.VideoPublishReq) (*contentpb.VideoPublishRes, error) {
	return nil, unimplemented()
}

func (s *contentDetailBenchServer) BackfillFollowInbox(context.Context, *contentpb.BackfillFollowInboxReq) (*contentpb.BackfillFollowInboxRes, error) {
	return nil, unimplemented()
}

func (s *contentDetailBenchServer) CleanupFollowInbox(context.Context, *contentpb.CleanupFollowInboxReq) (*contentpb.CleanupFollowInboxRes, error) {
	return nil, unimplemented()
}

func (s *contentDetailBenchServer) GetUploadCredentials(context.Context, *contentpb.GetUploadCredentialsReq) (*contentpb.GetUploadCredentialsRes, error) {
	return nil, unimplemented()
}

func (s *contentDetailBenchServer) EditArticle(context.Context, *contentpb.EditArticleReq) (*contentpb.EditArticleRes, error) {
	return nil, unimplemented()
}

func (s *contentDetailBenchServer) EditVideo(context.Context, *contentpb.EditVideoReq) (*contentpb.EditVideoRes, error) {
	return nil, unimplemented()
}

func (s *contentDetailBenchServer) DeleteContent(context.Context, *contentpb.DeleteContentReq) (*contentpb.DeleteContentRes, error) {
	return nil, unimplemented()
}

func unimplemented() error {
	return status.Error(codes.Unimplemented, "benchmark fixture only implements GetContentDetail")
}

func int64Ptr(value int64) *int64 {
	return &value
}
