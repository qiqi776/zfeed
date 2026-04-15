package search

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	searchpb "zfeed/app/rpc/search/search"
	searchservice "zfeed/app/rpc/search/searchservice"

	"google.golang.org/grpc"
)

type stubSearchService struct {
	searchUsersFunc    func(ctx context.Context, in *searchservice.SearchUsersReq, opts ...grpc.CallOption) (*searchservice.SearchUsersRes, error)
	searchContentsFunc func(ctx context.Context, in *searchservice.SearchContentsReq, opts ...grpc.CallOption) (*searchservice.SearchContentsRes, error)
}

func (s *stubSearchService) SearchUsers(ctx context.Context, in *searchservice.SearchUsersReq, opts ...grpc.CallOption) (*searchservice.SearchUsersRes, error) {
	return s.searchUsersFunc(ctx, in, opts...)
}

func (s *stubSearchService) SearchContents(ctx context.Context, in *searchservice.SearchContentsReq, opts ...grpc.CallOption) (*searchservice.SearchContentsRes, error) {
	return s.searchContentsFunc(ctx, in, opts...)
}

func TestSearchUsersReturnsFollowingState(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(2001))
	logic := NewSearchUsersLogic(ctx, &svc.ServiceContext{
		SearchRpc: &stubSearchService{
			searchUsersFunc: func(_ context.Context, in *searchservice.SearchUsersReq, _ ...grpc.CallOption) (*searchservice.SearchUsersRes, error) {
				if in.GetQuery() != "Ali" || in.GetViewerId() != 2001 {
					t.Fatalf("unexpected rpc request: %+v", in)
				}
				return &searchservice.SearchUsersRes{
					Items: []*searchpb.SearchUserItem{
						{UserId: 1001, Nickname: "Alice", Avatar: "a1", Bio: "growth notes", IsFollowing: false},
						{UserId: 1002, Nickname: "Alicia", Avatar: "a2", Bio: "design", IsFollowing: true},
					},
					NextCursor: 1002,
					HasMore:    false,
				}, nil
			},
		},
	})

	resp, err := logic.SearchUsers(&types.SearchUsersReq{
		Query:    stringPtr("Ali"),
		PageSize: uint32Ptr(10),
	})
	if err != nil {
		t.Fatalf("SearchUsers returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.Items))
	}
	if !resp.Items[1].IsFollowing {
		t.Fatal("expected second item to be following")
	}
}

func TestSearchContentsReturnsContentRows(t *testing.T) {
	logic := NewSearchContentsLogic(context.Background(), &svc.ServiceContext{
		SearchRpc: &stubSearchService{
			searchContentsFunc: func(_ context.Context, in *searchservice.SearchContentsReq, _ ...grpc.CallOption) (*searchservice.SearchContentsRes, error) {
				if in.GetQuery() != "Growth" {
					t.Fatalf("unexpected rpc request: %+v", in)
				}
				return &searchservice.SearchContentsRes{
					Items: []*searchpb.SearchContentItem{
						{
							ContentId:    4001,
							ContentType:  10,
							AuthorId:     3001,
							AuthorName:   "writer",
							AuthorAvatar: "avatar",
							Title:        "Growth Diary",
							CoverUrl:     "cover",
							PublishedAt:  1700000000,
						},
					},
					NextCursor: 0,
					HasMore:    false,
				}, nil
			},
		},
	})

	resp, err := logic.SearchContents(&types.SearchContentsReq{
		Query:    stringPtr("Growth"),
		PageSize: uint32Ptr(10),
	})
	if err != nil {
		t.Fatalf("SearchContents returned error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].ContentId != 4001 {
		t.Fatalf("content_id = %d, want 4001", resp.Items[0].ContentId)
	}
}

func stringPtr(value string) *string {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}
