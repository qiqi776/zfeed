package interaction

import (
	"context"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"

	"google.golang.org/grpc"
)

type queryReplyListService struct {
	fakeCommentService
	res *interactionpb.QueryReplyListRes
	err error
}

func (f *queryReplyListService) QueryReplyList(
	_ context.Context,
	in *interactionpb.QueryReplyListReq,
	_ ...grpc.CallOption,
) (*interactionpb.QueryReplyListRes, error) {
	f.queryReplyReq = in
	return f.res, f.err
}

func TestQueryReplyCommentListRejectsInvalidParams(t *testing.T) {
	validCommentID := int64(3001)
	cursor := int64(0)
	validPageSize := uint32(20)

	tests := []struct {
		name      string
		commentID int64
		pageSize  uint32
	}{
		{name: "zero comment", commentID: 0, pageSize: validPageSize},
		{name: "negative comment", commentID: -7, pageSize: validPageSize},
		{name: "zero page", commentID: validCommentID, pageSize: 0},
		{name: "oversize page", commentID: validCommentID, pageSize: 51},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRPC := &fakeCommentService{}
			logic := NewQueryReplyCommentListLogic(
				context.Background(),
				&svc.ServiceContext{CommentRpc: commentRPC},
			)

			resp, err := logic.QueryReplyCommentList(&types.QueryReplyCommentListReq{
				CommentId: &tt.commentID,
				Cursor:    &cursor,
				PageSize:  &tt.pageSize,
			})
			if err == nil {
				t.Fatal("QueryReplyCommentList returned nil error")
			}
			if resp != nil {
				t.Fatalf("QueryReplyCommentList response = %+v, want nil", resp)
			}
			if commentRPC.queryReplyReq != nil {
				t.Fatalf("CommentRpc.QueryReplyList was called with %+v", commentRPC.queryReplyReq)
			}
		})
	}
}

func TestQueryReplyCommentListMaps(t *testing.T) {
	commentID := int64(3001)
	cursor := int64(0)
	pageSize := uint32(20)
	commentRPC := &queryReplyListService{
		res: &interactionpb.QueryReplyListRes{
			Replies: []*interactionpb.CommentItem{
				nil,
				{
					CommentId:     4001,
					ContentId:     2001,
					UserId:        5001,
					ReplyToUserId: 6001,
					ParentId:      3001,
					RootId:        commentID,
					Comment:       "reply",
					CreatedAt:     1710000100,
					Status:        1,
					UserName:      "bob",
					UserAvatar:    "https://cdn.example/bob.png",
					ReplyCount:    3,
				},
			},
			NextCursor: 4001,
			HasMore:    true,
		},
	}
	logic := NewQueryReplyCommentListLogic(
		context.Background(),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.QueryReplyCommentList(&types.QueryReplyCommentListReq{
		CommentId: &commentID,
		Cursor:    &cursor,
		PageSize:  &pageSize,
	})
	if err != nil {
		t.Fatalf("QueryReplyCommentList returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("QueryReplyCommentList returned nil response")
	}
	if commentRPC.queryReplyReq == nil {
		t.Fatal("CommentRpc.QueryReplyList was not called")
	}
	if resp.NextCursor != 4001 || !resp.HasMore {
		t.Fatalf("pagination = (%d,%v), want (4001,true)", resp.NextCursor, resp.HasMore)
	}
	if len(resp.Comments) != 1 {
		t.Fatalf("comments length = %d, want 1", len(resp.Comments))
	}
	got := resp.Comments[0]
	if got.CommentId != 4001 || got.ContentId != 2001 || got.UserId != 5001 ||
		got.ReplyToUserId != 6001 || got.ParentId != 3001 || got.RootId != commentID ||
		got.Comment != "reply" || got.CreatedAt != 1710000100 || got.Status != 1 ||
		got.UserName != "bob" || got.UserAvatar != "https://cdn.example/bob.png" ||
		got.ReplyCount != 3 {
		t.Fatalf("reply item = %+v", got)
	}
}

func TestQueryReplyCommentListNilRPC(t *testing.T) {
	commentID := int64(3001)
	cursor := int64(0)
	pageSize := uint32(20)
	commentRPC := &queryReplyListService{}
	logic := NewQueryReplyCommentListLogic(
		context.Background(),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.QueryReplyCommentList(&types.QueryReplyCommentListReq{
		CommentId: &commentID,
		Cursor:    &cursor,
		PageSize:  &pageSize,
	})
	if err == nil {
		t.Fatal("QueryReplyCommentList returned nil error")
	}
	if resp != nil {
		t.Fatalf("QueryReplyCommentList response = %+v, want nil", resp)
	}
}
