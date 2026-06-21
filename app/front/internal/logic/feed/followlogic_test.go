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

type followFeedRPC struct {
	fakeFeedRPC
	req *contentpb.FollowFeedReq
	res *contentpb.FollowFeedRes
	err error
}

func (f *followFeedRPC) FollowFeed(
	_ context.Context,
	in *contentpb.FollowFeedReq,
	_ ...grpc.CallOption,
) (*contentpb.FollowFeedRes, error) {
	f.req = in
	return f.res, f.err
}

func TestFollowMaps(t *testing.T) {
	cursor := "follow:12345"
	pageSize := uint32(20)
	feedRPC := &followFeedRPC{
		res: &contentpb.FollowFeedRes{
			Items: []*contentpb.FollowFeedItem{
				nil,
				{
					ContentId:    2001,
					ContentType:  contentpb.ContentType_VIDEO,
					AuthorId:     1002,
					AuthorName:   "bob",
					AuthorAvatar: "https://cdn.example/bob.png",
					Title:        "video",
					CoverUrl:     "https://cdn.example/video.png",
					PublishedAt:  123456,
					IsLiked:      true,
					LikeCount:    11,
				},
			},
			NextCursor: "follow:777",
			HasMore:    true,
		},
	}
	logic := NewFollowLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.Follow(&types.FollowFeedReq{
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if err != nil {
		t.Fatalf("Follow returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Follow returned nil response")
	}
	if feedRPC.req == nil {
		t.Fatal("FeedRpc.FollowFeed was not called")
	}
	if feedRPC.req.GetUserId() != 1001 || feedRPC.req.GetCursor() != cursor ||
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
		item.Title != "video" || item.CoverUrl != "https://cdn.example/video.png" ||
		item.PublishedAt != 123456 || !item.IsLiked || item.LikeCount != 11 {
		t.Fatalf("follow item = %+v", item)
	}
	if resp.NextCursor != "follow:777" || !resp.HasMore {
		t.Fatalf("Follow response = %+v", resp)
	}
}

func TestFollowRPCError(t *testing.T) {
	cursor := "follow:12345"
	pageSize := uint32(20)
	rpcErr := errors.New("feed rpc down")
	feedRPC := &followFeedRPC{err: rpcErr}
	logic := NewFollowLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.Follow(&types.FollowFeedReq{
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("Follow error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("Follow response = %+v, want nil", resp)
	}
}

func TestFollowNilRPC(t *testing.T) {
	cursor := "follow:12345"
	pageSize := uint32(20)
	feedRPC := &followFeedRPC{}
	logic := NewFollowLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.Follow(&types.FollowFeedReq{
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if err == nil {
		t.Fatal("Follow returned nil error")
	}
	if resp != nil {
		t.Fatalf("Follow response = %+v, want nil", resp)
	}
}
