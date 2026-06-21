package feed

import (
	"context"
	"errors"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"

	"google.golang.org/grpc"
)

type userPublishFeedRPC struct {
	fakeFeedRPC
	req *contentpb.UserPublishFeedReq
	res *contentpb.UserPublishFeedRes
	err error
}

func (f *userPublishFeedRPC) UserPublishFeed(
	_ context.Context,
	in *contentpb.UserPublishFeedReq,
	_ ...grpc.CallOption,
) (*contentpb.UserPublishFeedRes, error) {
	f.req = in
	return f.res, f.err
}

func TestUserPublishMaps(t *testing.T) {
	userID := int64(1002)
	reqViewerID := int64(3003)
	cursor := "publish:12345"
	pageSize := uint32(20)
	feedRPC := &userPublishFeedRPC{
		res: &contentpb.UserPublishFeedRes{
			Items: []*contentpb.ContentItem{
				nil,
				{
					ContentId:    2001,
					ContentType:  contentpb.ContentType_VIDEO,
					AuthorId:     1002,
					AuthorName:   "bob",
					AuthorAvatar: "https://cdn.example/bob.png",
					Title:        "publish video",
					CoverUrl:     "https://cdn.example/publish.png",
					PublishedAt:  123456,
					IsLiked:      true,
					LikeCount:    17,
				},
			},
			NextCursor: "publish:777",
			HasMore:    true,
		},
	}
	logic := NewUserPublishLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.UserPublish(&types.UserPublishFeedReq{
		ViewerId: &reqViewerID,
		UserId:   &userID,
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if err != nil {
		t.Fatalf("UserPublish returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("UserPublish returned nil response")
	}
	if feedRPC.req == nil {
		t.Fatal("FeedRpc.UserPublishFeed was not called")
	}
	if feedRPC.req.ViewerId == nil || feedRPC.req.GetViewerId() != 1001 ||
		feedRPC.req.GetAuthorId() != userID || feedRPC.req.GetCursor() != cursor ||
		feedRPC.req.GetPageSize() != pageSize {
		t.Fatalf("rpc request = %+v", feedRPC.req)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.Items))
	}
	item := resp.Items[0]
	if item.ContentId != 2001 || item.ContentType != int32(contentpb.ContentType_VIDEO) ||
		item.AuthorId != 1002 || item.AuthorName != "bob" ||
		item.AuthorAvatar != "https://cdn.example/bob.png" ||
		item.Title != "publish video" || item.CoverUrl != "https://cdn.example/publish.png" ||
		item.PublishedAt != 123456 || !item.IsLiked || item.LikeCount != 17 {
		t.Fatalf("publish item = %+v", item)
	}
	if resp.NextCursor != "publish:777" || !resp.HasMore {
		t.Fatalf("UserPublish response = %+v", resp)
	}
}

func TestUserPublishRPCError(t *testing.T) {
	userID := int64(1002)
	viewerID := int64(3003)
	cursor := "publish:12345"
	pageSize := uint32(20)
	rpcErr := errors.New("publish feed rpc down")
	feedRPC := &userPublishFeedRPC{err: rpcErr}
	logic := NewUserPublishLogic(
		context.Background(),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.UserPublish(&types.UserPublishFeedReq{
		ViewerId: &viewerID,
		UserId:   &userID,
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("UserPublish error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("UserPublish response = %+v, want nil", resp)
	}
	if feedRPC.req == nil || feedRPC.req.ViewerId == nil || feedRPC.req.GetViewerId() != viewerID {
		t.Fatalf("rpc request = %+v", feedRPC.req)
	}
}

func TestUserPublishNilRPC(t *testing.T) {
	userID := int64(1002)
	cursor := "publish:12345"
	pageSize := uint32(20)
	feedRPC := &userPublishFeedRPC{}
	logic := NewUserPublishLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.UserPublish(&types.UserPublishFeedReq{
		UserId:   &userID,
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if err == nil {
		t.Fatal("UserPublish returned nil error")
	}
	if resp != nil {
		t.Fatalf("UserPublish response = %+v, want nil", resp)
	}
}
