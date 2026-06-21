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

type recommendFeedRPC struct {
	fakeFeedRPC
	req *contentpb.RecommendFeedReq
	res *contentpb.RecommendFeedRes
	err error
}

func (f *recommendFeedRPC) RecommendFeed(
	_ context.Context,
	in *contentpb.RecommendFeedReq,
	_ ...grpc.CallOption,
) (*contentpb.RecommendFeedRes, error) {
	f.req = in
	return f.res, f.err
}

func TestRecommendMaps(t *testing.T) {
	cursor := "12345"
	pageSize := uint32(20)
	snapshotID := "rec:0001:b:hash:1"
	feedRPC := &recommendFeedRPC{
		res: &contentpb.RecommendFeedRes{
			Items: []*contentpb.ContentItem{
				nil,
				{
					ContentId:    2001,
					ContentType:  contentpb.ContentType_ARTICLE,
					AuthorId:     1002,
					AuthorName:   "alice",
					AuthorAvatar: "https://cdn.example/avatar.png",
					Title:        "hello",
					CoverUrl:     "https://cdn.example/cover.png",
					PublishedAt:  123456,
					IsLiked:      true,
					LikeCount:    9,
				},
			},
			NextCursor: 777,
			HasMore:    true,
			SnapshotId: "rec:next",
		},
	}
	logic := NewRecommendLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.Recommend(&types.RecommendFeedReq{
		Cursor:     &cursor,
		PageSize:   &pageSize,
		SnapshotId: &snapshotID,
	})
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Recommend returned nil response")
	}
	if feedRPC.req == nil {
		t.Fatal("FeedRpc.RecommendFeed was not called")
	}
	if feedRPC.req.GetUserId() != 1001 || feedRPC.req.GetCursor() != cursor ||
		feedRPC.req.GetPageSize() != pageSize || feedRPC.req.GetSnapshotId() != snapshotID {
		t.Fatalf("rpc request = %+v", feedRPC.req)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.Items))
	}
	item := resp.Items[0]
	if item.ContentId != 2001 || item.ContentType != int32(contentpb.ContentType_ARTICLE) ||
		item.AuthorId != 1002 || item.AuthorName != "alice" ||
		item.AuthorAvatar != "https://cdn.example/avatar.png" ||
		item.Title != "hello" || item.CoverUrl != "https://cdn.example/cover.png" ||
		item.PublishedAt != 123456 || !item.IsLiked || item.LikeCount != 9 {
		t.Fatalf("recommend item = %+v", item)
	}
	if resp.NextCursor != "777" || !resp.HasMore || resp.SnapshotId != "rec:next" {
		t.Fatalf("Recommend response = %+v", resp)
	}
}

func TestRecommendRPCError(t *testing.T) {
	cursor := "12345"
	pageSize := uint32(20)
	rpcErr := errors.New("feed rpc down")
	feedRPC := &recommendFeedRPC{err: rpcErr}
	logic := NewRecommendLogic(
		context.Background(),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.Recommend(&types.RecommendFeedReq{
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("Recommend error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("Recommend response = %+v, want nil", resp)
	}
}

func TestRecommendNilRPC(t *testing.T) {
	cursor := "12345"
	pageSize := uint32(20)
	feedRPC := &recommendFeedRPC{}
	logic := NewRecommendLogic(
		context.Background(),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.Recommend(&types.RecommendFeedReq{
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if err == nil {
		t.Fatal("Recommend returned nil error")
	}
	if resp != nil {
		t.Fatalf("Recommend response = %+v, want nil", resp)
	}
}
