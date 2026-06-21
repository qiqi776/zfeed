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

type userFavoriteFeedRPC struct {
	fakeFeedRPC
	req *contentpb.UserFavoriteFeedReq
	res *contentpb.UserFavoriteFeedRes
	err error
}

func (f *userFavoriteFeedRPC) UserFavoriteFeed(
	_ context.Context,
	in *contentpb.UserFavoriteFeedReq,
	_ ...grpc.CallOption,
) (*contentpb.UserFavoriteFeedRes, error) {
	f.req = in
	return f.res, f.err
}

func TestUserFavoriteMaps(t *testing.T) {
	userID := int64(1002)
	cursor := "favorite:12345"
	pageSize := uint32(20)
	feedRPC := &userFavoriteFeedRPC{
		res: &contentpb.UserFavoriteFeedRes{
			Items: []*contentpb.ContentItem{
				nil,
				{
					ContentId:    2001,
					ContentType:  contentpb.ContentType_ARTICLE,
					AuthorId:     1002,
					AuthorName:   "alice",
					AuthorAvatar: "https://cdn.example/alice.png",
					Title:        "favorite article",
					CoverUrl:     "https://cdn.example/favorite.png",
					PublishedAt:  123456,
					IsLiked:      true,
					LikeCount:    13,
				},
			},
			NextCursor: "favorite:777",
			HasMore:    true,
		},
	}
	logic := NewUserFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.UserFavorite(&types.UserFavoriteFeedReq{
		UserId:   &userID,
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if err != nil {
		t.Fatalf("UserFavorite returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("UserFavorite returned nil response")
	}
	if feedRPC.req == nil {
		t.Fatal("FeedRpc.UserFavoriteFeed was not called")
	}
	if feedRPC.req.ViewerId == nil || feedRPC.req.GetViewerId() != 1001 ||
		feedRPC.req.GetUserId() != userID || feedRPC.req.GetCursor() != cursor ||
		feedRPC.req.GetPageSize() != pageSize {
		t.Fatalf("rpc request = %+v", feedRPC.req)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.Items))
	}
	item := resp.Items[0]
	if item.ContentId != 2001 || item.ContentType != int32(contentpb.ContentType_ARTICLE) ||
		item.AuthorId != 1002 || item.AuthorName != "alice" ||
		item.AuthorAvatar != "https://cdn.example/alice.png" ||
		item.Title != "favorite article" || item.CoverUrl != "https://cdn.example/favorite.png" ||
		item.PublishedAt != 123456 || !item.IsLiked || item.LikeCount != 13 {
		t.Fatalf("favorite item = %+v", item)
	}
	if resp.NextCursor != "favorite:777" || !resp.HasMore {
		t.Fatalf("UserFavorite response = %+v", resp)
	}
}

func TestUserFavoriteRPCError(t *testing.T) {
	userID := int64(1002)
	cursor := "favorite:12345"
	pageSize := uint32(20)
	rpcErr := errors.New("favorite feed rpc down")
	feedRPC := &userFavoriteFeedRPC{err: rpcErr}
	logic := NewUserFavoriteLogic(
		context.Background(),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.UserFavorite(&types.UserFavoriteFeedReq{
		UserId:   &userID,
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("UserFavorite error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("UserFavorite response = %+v, want nil", resp)
	}
	if feedRPC.req == nil || feedRPC.req.ViewerId != nil {
		t.Fatalf("rpc request = %+v", feedRPC.req)
	}
}

func TestUserFavoriteNilRPC(t *testing.T) {
	userID := int64(1002)
	cursor := "favorite:12345"
	pageSize := uint32(20)
	feedRPC := &userFavoriteFeedRPC{}
	logic := NewUserFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{FeedRpc: feedRPC},
	)

	resp, err := logic.UserFavorite(&types.UserFavoriteFeedReq{
		UserId:   &userID,
		Cursor:   &cursor,
		PageSize: &pageSize,
	})
	if err == nil {
		t.Fatal("UserFavorite returned nil error")
	}
	if resp != nil {
		t.Fatalf("UserFavorite response = %+v, want nil", resp)
	}
}
