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

type deleteCommentService struct {
	fakeCommentService
	res *interactionpb.DeleteCommentRes
	err error
}

func (f *deleteCommentService) DeleteComment(
	_ context.Context,
	in *interactionpb.DeleteCommentReq,
	_ ...grpc.CallOption,
) (*interactionpb.DeleteCommentRes, error) {
	f.deleteCommentReq = in
	return f.res, f.err
}

func TestDeleteCommentRejectsInvalidID(t *testing.T) {
	validCommentID := int64(3001)
	validContentID := int64(2001)
	scene := "ARTICLE"

	tests := []struct {
		name      string
		commentID int64
		contentID int64
	}{
		{name: "zero comment", commentID: 0, contentID: validContentID},
		{name: "negative comment", commentID: -7, contentID: validContentID},
		{name: "zero content", commentID: validCommentID, contentID: 0},
		{name: "negative content", commentID: validCommentID, contentID: -7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRPC := &fakeCommentService{}
			logic := NewDeleteCommentLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{CommentRpc: commentRPC},
			)

			resp, err := logic.DeleteComment(&types.DeleteCommentReq{
				CommentId: &tt.commentID,
				ContentId: &tt.contentID,
				Scene:     &scene,
			})
			if err == nil {
				t.Fatal("DeleteComment returned nil error")
			}
			if resp != nil {
				t.Fatalf("DeleteComment response = %+v, want nil", resp)
			}
			if commentRPC.deleteCommentReq != nil {
				t.Fatalf("CommentRpc.DeleteComment was called with %+v", commentRPC.deleteCommentReq)
			}
		})
	}
}

func TestDeleteCommentRejectsInvalidThread(t *testing.T) {
	commentID := int64(3001)
	contentID := int64(2001)
	negativeID := int64(-1)
	scene := "ARTICLE"

	tests := []struct {
		name   string
		mutate func(*types.DeleteCommentReq)
	}{
		{
			name: "negative root",
			mutate: func(req *types.DeleteCommentReq) {
				req.RootId = &negativeID
			},
		},
		{
			name: "negative parent",
			mutate: func(req *types.DeleteCommentReq) {
				req.ParentId = &negativeID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentRPC := &fakeCommentService{}
			logic := NewDeleteCommentLogic(
				context.WithValue(context.Background(), "user_id", int64(1001)),
				&svc.ServiceContext{CommentRpc: commentRPC},
			)
			req := &types.DeleteCommentReq{
				CommentId: &commentID,
				ContentId: &contentID,
				Scene:     &scene,
			}
			tt.mutate(req)

			resp, err := logic.DeleteComment(req)
			if err == nil {
				t.Fatal("DeleteComment returned nil error")
			}
			if resp != nil {
				t.Fatalf("DeleteComment response = %+v, want nil", resp)
			}
			if commentRPC.deleteCommentReq != nil {
				t.Fatalf("CommentRpc.DeleteComment was called with %+v", commentRPC.deleteCommentReq)
			}
		})
	}
}

func TestDeleteCommentMaps(t *testing.T) {
	commentID := int64(3001)
	contentID := int64(2001)
	rootID := int64(3000)
	parentID := int64(3000)
	scene := "ARTICLE"
	commentRPC := &deleteCommentService{res: &interactionpb.DeleteCommentRes{}}
	logic := NewDeleteCommentLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.DeleteComment(&types.DeleteCommentReq{
		CommentId: &commentID,
		ContentId: &contentID,
		RootId:    &rootID,
		ParentId:  &parentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("DeleteComment returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("DeleteComment returned nil response")
	}
	if commentRPC.deleteCommentReq == nil {
		t.Fatal("CommentRpc.DeleteComment was not called")
	}
	req := commentRPC.deleteCommentReq
	if req.GetUserId() != 1001 || req.GetCommentId() != commentID || req.GetContentId() != contentID ||
		req.GetRootId() != rootID || req.GetParentId() != parentID || req.GetScene() != interactionpb.Scene_ARTICLE {
		t.Fatalf("rpc request = %+v", req)
	}
}

func TestDeleteCommentRPCError(t *testing.T) {
	commentID := int64(3001)
	contentID := int64(2001)
	scene := "ARTICLE"
	rpcErr := errors.New("comment rpc down")
	commentRPC := &deleteCommentService{err: rpcErr}
	logic := NewDeleteCommentLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.DeleteComment(&types.DeleteCommentReq{
		CommentId: &commentID,
		ContentId: &contentID,
		Scene:     &scene,
	})
	if !errors.Is(err, rpcErr) {
		t.Fatalf("DeleteComment error = %v, want %v", err, rpcErr)
	}
	if resp != nil {
		t.Fatalf("DeleteComment response = %+v, want nil", resp)
	}
}

func TestDeleteCommentNilRPC(t *testing.T) {
	commentID := int64(3001)
	contentID := int64(2001)
	scene := "ARTICLE"
	commentRPC := &deleteCommentService{}
	logic := NewDeleteCommentLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{CommentRpc: commentRPC},
	)

	resp, err := logic.DeleteComment(&types.DeleteCommentReq{
		CommentId: &commentID,
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err == nil {
		t.Fatal("DeleteComment returned nil error")
	}
	if resp != nil {
		t.Fatalf("DeleteComment response = %+v, want nil", resp)
	}
}
