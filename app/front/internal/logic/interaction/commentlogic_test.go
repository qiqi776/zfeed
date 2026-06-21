package interaction

import (
	"context"
	"errors"
	"testing"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"

	"google.golang.org/grpc"
)

type commentService struct {
	fakeCommentService
	res *interactionpb.CommentRes
	err error
}

func (f *commentService) Comment(
	_ context.Context,
	in *interactionpb.CommentReq,
	_ ...grpc.CallOption,
) (*interactionpb.CommentRes, error) {
	f.commentReq = in
	return f.res, f.err
}

func TestCommentRejectsBlank(t *testing.T) {
	contentID := int64(2001)
	contentUserID := int64(3001)
	scene := "ARTICLE"

	tests := []struct {
		name    string
		comment string
	}{
		{name: "empty", comment: ""},
		{name: "spaces", comment: " \t\n "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRPC := &fakeCommentService{}
			logic := NewCommentLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{
					CommentRpc: commentRPC,
				},
			)

			resp, err := logic.Comment(&types.CommentReq{
				ContentId:     &contentID,
				ContentUserId: &contentUserID,
				Scene:         &scene,
				Comment:       &tt.comment,
			})
			if err == nil {
				t.Fatal("Comment returned nil error")
			}
			if resp != nil {
				t.Fatalf("Comment response = %+v, want nil", resp)
			}
			if commentRPC.commentReq != nil {
				t.Fatalf("CommentRpc.Comment was called with %+v", commentRPC.commentReq)
			}
		})
	}
}

func TestCommentRejectsInvalidID(t *testing.T) {
	validContentID := int64(2001)
	validContentUserID := int64(3001)
	scene := "ARTICLE"
	comment := "nice post"

	tests := []struct {
		name          string
		contentID     int64
		contentUserID int64
	}{
		{name: "zero content", contentID: 0, contentUserID: validContentUserID},
		{name: "negative content", contentID: -7, contentUserID: validContentUserID},
		{name: "zero content user", contentID: validContentID, contentUserID: 0},
		{name: "negative content user", contentID: validContentID, contentUserID: -7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRPC := &fakeCommentService{}
			logic := NewCommentLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{CommentRpc: commentRPC},
			)

			resp, err := logic.Comment(&types.CommentReq{
				ContentId:     &tt.contentID,
				ContentUserId: &tt.contentUserID,
				Scene:         &scene,
				Comment:       &comment,
			})
			if err == nil {
				t.Fatal("Comment returned nil error")
			}
			if resp != nil {
				t.Fatalf("Comment response = %+v, want nil", resp)
			}
			if commentRPC.commentReq != nil {
				t.Fatalf("CommentRpc.Comment was called with %+v", commentRPC.commentReq)
			}
		})
	}
}

func TestCommentRejectsInvalidThread(t *testing.T) {
	contentID := int64(2001)
	contentUserID := int64(3001)
	negativeID := int64(-1)
	scene := "ARTICLE"
	comment := "nice post"

	tests := []struct {
		name   string
		mutate func(*types.CommentReq)
	}{
		{
			name: "negative parent",
			mutate: func(req *types.CommentReq) {
				req.ParentId = &negativeID
			},
		},
		{
			name: "negative root",
			mutate: func(req *types.CommentReq) {
				req.RootId = &negativeID
			},
		},
		{
			name: "negative reply user",
			mutate: func(req *types.CommentReq) {
				req.ReplyToUserId = &negativeID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRPC := &fakeCommentService{}
			logic := NewCommentLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{CommentRpc: commentRPC},
			)
			req := &types.CommentReq{
				ContentId:     &contentID,
				ContentUserId: &contentUserID,
				Scene:         &scene,
				Comment:       &comment,
			}
			tt.mutate(req)

			resp, err := logic.Comment(req)
			if err == nil {
				t.Fatal("Comment returned nil error")
			}
			if resp != nil {
				t.Fatalf("Comment response = %+v, want nil", resp)
			}
			if commentRPC.commentReq != nil {
				t.Fatalf("CommentRpc.Comment was called with %+v", commentRPC.commentReq)
			}
		})
	}
}

func TestCommentMaps(t *testing.T) {
	contentID := int64(2001)
	contentUserID := int64(3001)
	parentID := int64(4001)
	rootID := int64(5001)
	replyToUserID := int64(6001)
	scene := " article "
	comment := "  nice post  "
	commentRPC := &commentService{res: &interactionpb.CommentRes{CommentId: 7001}}
	logic := NewCommentLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.Comment(&types.CommentReq{
		ContentId:     &contentID,
		ContentUserId: &contentUserID,
		Scene:         &scene,
		Comment:       &comment,
		ParentId:      &parentID,
		RootId:        &rootID,
		ReplyToUserId: &replyToUserID,
	})
	if err != nil {
		t.Fatalf("Comment returned error: %v", err)
	}
	if resp == nil || resp.CommentId != 7001 {
		t.Fatalf("Comment response = %+v", resp)
	}
	if commentRPC.commentReq == nil {
		t.Fatal("CommentRpc.Comment was not called")
	}
	req := commentRPC.commentReq
	if req.GetUserId() != 1001 || req.GetContentId() != contentID ||
		req.GetContentUserId() != contentUserID || req.GetScene() != interactionpb.Scene_ARTICLE ||
		req.GetComment() != "nice post" || req.GetParentId() != parentID ||
		req.GetRootId() != rootID || req.GetReplyToUserId() != replyToUserID {
		t.Fatalf("rpc request = %+v", req)
	}
}

func TestCommentRPCError(t *testing.T) {
	contentID := int64(2001)
	contentUserID := int64(3001)
	scene := "ARTICLE"
	comment := "nice post"
	rpcErr := errors.New("comment rpc down")
	commentRPC := &commentService{err: rpcErr}
	logic := NewCommentLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.Comment(&types.CommentReq{
		ContentId:     &contentID,
		ContentUserId: &contentUserID,
		Scene:         &scene,
		Comment:       &comment,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("Comment error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("Comment response = %+v, want nil", resp)
	}
}

func TestCommentNilRPC(t *testing.T) {
	contentID := int64(2001)
	contentUserID := int64(3001)
	scene := "ARTICLE"
	comment := "nice post"
	commentRPC := &commentService{}
	logic := NewCommentLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.Comment(&types.CommentReq{
		ContentId:     &contentID,
		ContentUserId: &contentUserID,
		Scene:         &scene,
		Comment:       &comment,
	})
	if err == nil {
		t.Fatal("Comment returned nil error")
	}
	if resp != nil {
		t.Fatalf("Comment response = %+v, want nil", resp)
	}
}
