package content

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
)

func TestContentUploadsCredentialsCallsContentRPC(t *testing.T) {
	logic := NewContentUploadsCredentialsLogic(context.Background(), &svc.ServiceContext{
		ContentRpc: &fakeContentService{
			uploadCredsFunc: func(_ context.Context, in *contentpb.GetUploadCredentialsReq, _ ...grpc.CallOption) (*contentpb.GetUploadCredentialsRes, error) {
				if in.GetScene() != "article-cover" || in.GetFileExt() != ".png" || in.GetFileSize() != 2048 || in.GetFileName() != "cover.png" {
					t.Fatalf("unexpected upload credentials request: %+v", in)
				}
				return &contentpb.GetUploadCredentialsRes{
					ObjectKey: "zfeed/article-cover/2026/04/14/test.png",
					Url:       "https://cdn.example.com/zfeed/article-cover/2026/04/14/test.png",
					ExpiredAt: 1_776_155_400,
					FormData: &contentpb.OssFormData{
						Host:             "https://oss-cn-hangzhou.aliyuncs.com",
						Policy:           "policy",
						Signature:        "signature",
						SecurityToken:    "token",
						SignatureVersion: "OSS4-HMAC-SHA256",
						Credential:       "credential",
						Date:             "20260414T083000Z",
						Key:              "zfeed/article-cover/2026/04/14/test.png",
					},
				}, nil
			},
		},
	})

	resp, err := logic.ContentUploadsCredentials(&types.ContentUploadsCredentialsReq{
		Scene:    strPtr("article-cover"),
		FileExt:  strPtr(".png"),
		FileSize: int64Ptr(2048),
		FileName: strPtr("cover.png"),
	})
	if err != nil {
		t.Fatalf("ContentUploadsCredentials returned error: %v", err)
	}
	if resp.ObjectKey != "zfeed/article-cover/2026/04/14/test.png" || resp.FormData.SignatureVersion != "OSS4-HMAC-SHA256" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestContentUploadsCredentialsRejectsInvalidRequest(t *testing.T) {
	logic := NewContentUploadsCredentialsLogic(context.Background(), &svc.ServiceContext{})

	if _, err := logic.ContentUploadsCredentials(&types.ContentUploadsCredentialsReq{}); err == nil {
		t.Fatal("expected invalid request error")
	}
}
