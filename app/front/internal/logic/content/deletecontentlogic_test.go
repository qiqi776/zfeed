package content

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
)

func TestDeleteContentLogicCallsContentRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(7))
	logic := NewDeleteContentLogic(ctx, &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			deleteContentFunc: func(_ context.Context, in *contentpb.DeleteContentReq, _ ...grpc.CallOption) (*contentpb.DeleteContentRes, error) {
				if in.GetUserId() != 7 || in.GetContentId() != 101 {
					t.Fatalf("unexpected delete request: %+v", in)
				}
				return &contentpb.DeleteContentRes{}, nil
			},
		},
	})

	if _, err := logic.DeleteContent(&types.DeleteContentReq{ContentId: 101}); err != nil {
		t.Fatalf("DeleteContent returned error: %v", err)
	}
}
