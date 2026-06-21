package interaction

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"

	"google.golang.org/grpc"
)

type queryCommentListService struct {
	fakeCommentService
	res *interactionpb.QueryCommentListRes
	err error
}

func (f *queryCommentListService) QueryCommentList(
	_ context.Context,
	in *interactionpb.QueryCommentListReq,
	_ ...grpc.CallOption,
) (*interactionpb.QueryCommentListRes, error) {
	f.queryCommentReq = in
	return f.res, f.err
}

func TestQueryCommentListRejectsInvalidParams(t *testing.T) {
	validContentID := int64(2001)
	cursor := int64(0)
	validPageSize := uint32(20)
	scene := "ARTICLE"

	tests := []struct {
		name      string
		contentID int64
		pageSize  uint32
	}{
		{name: "zero content", contentID: 0, pageSize: validPageSize},
		{name: "negative content", contentID: -7, pageSize: validPageSize},
		{name: "zero page", contentID: validContentID, pageSize: 0},
		{name: "oversize page", contentID: validContentID, pageSize: 51},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRPC := &fakeCommentService{}
			logic := NewQueryCommentListLogic(
				context.Background(),
				&svc.ServiceContext{CommentRpc: commentRPC},
			)

			resp, err := logic.QueryCommentList(&types.QueryCommentListReq{
				ContentId: &tt.contentID,
				Scene:     &scene,
				Cursor:    &cursor,
				PageSize:  &tt.pageSize,
			})
			if err == nil {
				t.Fatal("QueryCommentList returned nil error")
			}
			if resp != nil {
				t.Fatalf("QueryCommentList response = %+v, want nil", resp)
			}
			if commentRPC.queryCommentReq != nil {
				t.Fatalf("CommentRpc.QueryCommentList was called with %+v", commentRPC.queryCommentReq)
			}
		})
	}
}

func TestQueryCommentListMaps(t *testing.T) {
	contentID := int64(2001)
	cursor := int64(0)
	pageSize := uint32(20)
	scene := "ARTICLE"
	commentRPC := &queryCommentListService{
		res: &interactionpb.QueryCommentListRes{
			Comments: []*interactionpb.CommentItem{
				nil,
				{
					CommentId:     3001,
					ContentId:     contentID,
					UserId:        4001,
					ReplyToUserId: 5001,
					ParentId:      6001,
					RootId:        7001,
					Comment:       "hello",
					CreatedAt:     1710000000,
					Status:        1,
					UserName:      "alice",
					UserAvatar:    "https://cdn.example/avatar.png",
					ReplyCount:    8,
				},
			},
			NextCursor: 3001,
			HasMore:    true,
		},
	}
	logic := NewQueryCommentListLogic(
		context.Background(),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.QueryCommentList(&types.QueryCommentListReq{
		ContentId: &contentID,
		Scene:     &scene,
		Cursor:    &cursor,
		PageSize:  &pageSize,
	})
	if err != nil {
		t.Fatalf("QueryCommentList returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("QueryCommentList returned nil response")
	}
	if commentRPC.queryCommentReq == nil {
		t.Fatal("CommentRpc.QueryCommentList was not called")
	}
	if resp.NextCursor != 3001 || !resp.HasMore {
		t.Fatalf("pagination = (%d,%v), want (3001,true)", resp.NextCursor, resp.HasMore)
	}
	if len(resp.Comments) != 1 {
		t.Fatalf("comments length = %d, want 1", len(resp.Comments))
	}
	got := resp.Comments[0]
	if got.CommentId != 3001 || got.ContentId != contentID || got.UserId != 4001 ||
		got.ReplyToUserId != 5001 || got.ParentId != 6001 || got.RootId != 7001 ||
		got.Comment != "hello" || got.CreatedAt != 1710000000 || got.Status != 1 ||
		got.UserName != "alice" || got.UserAvatar != "https://cdn.example/avatar.png" ||
		got.ReplyCount != 8 {
		t.Fatalf("comment item = %+v", got)
	}
}

func TestQueryCommentListNilRPC(t *testing.T) {
	contentID := int64(2001)
	cursor := int64(0)
	pageSize := uint32(20)
	scene := "ARTICLE"
	commentRPC := &queryCommentListService{}
	logic := NewQueryCommentListLogic(
		context.Background(),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.QueryCommentList(&types.QueryCommentListReq{
		ContentId: &contentID,
		Scene:     &scene,
		Cursor:    &cursor,
		PageSize:  &pageSize,
	})
	if err == nil {
		t.Fatal("QueryCommentList returned nil error")
	}
	if resp != nil {
		t.Fatalf("QueryCommentList response = %+v, want nil", resp)
	}
}
