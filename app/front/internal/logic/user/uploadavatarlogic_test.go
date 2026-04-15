package user

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	contentpb "zfeed/app/rpc/content/content"
	contentservice "zfeed/app/rpc/content/contentservice"
)

type avatarContentServiceStub struct {
	getUploadCredentialsFunc func(ctx context.Context, in *contentservice.GetUploadCredentialsReq, opts ...grpc.CallOption) (*contentservice.GetUploadCredentialsRes, error)
}

func (s *avatarContentServiceStub) PublishArticle(context.Context, *contentservice.ArticlePublishReq, ...grpc.CallOption) (*contentservice.ArticlePublishRes, error) {
	return nil, errors.New("unexpected PublishArticle call")
}

func (s *avatarContentServiceStub) PublishVideo(context.Context, *contentservice.VideoPublishReq, ...grpc.CallOption) (*contentservice.VideoPublishRes, error) {
	return nil, errors.New("unexpected PublishVideo call")
}

func (s *avatarContentServiceStub) BackfillFollowInbox(context.Context, *contentservice.BackfillFollowInboxReq, ...grpc.CallOption) (*contentservice.BackfillFollowInboxRes, error) {
	return nil, errors.New("unexpected BackfillFollowInbox call")
}

func (s *avatarContentServiceStub) GetUploadCredentials(ctx context.Context, in *contentservice.GetUploadCredentialsReq, opts ...grpc.CallOption) (*contentservice.GetUploadCredentialsRes, error) {
	if s.getUploadCredentialsFunc == nil {
		return nil, errors.New("unexpected GetUploadCredentials call")
	}
	return s.getUploadCredentialsFunc(ctx, in, opts...)
}

func (s *avatarContentServiceStub) GetContentDetail(context.Context, *contentservice.GetContentDetailReq, ...grpc.CallOption) (*contentservice.GetContentDetailRes, error) {
	return nil, errors.New("unexpected GetContentDetail call")
}

func (s *avatarContentServiceStub) EditArticle(context.Context, *contentservice.EditArticleReq, ...grpc.CallOption) (*contentservice.EditArticleRes, error) {
	return nil, errors.New("unexpected EditArticle call")
}

func (s *avatarContentServiceStub) EditVideo(context.Context, *contentservice.EditVideoReq, ...grpc.CallOption) (*contentservice.EditVideoRes, error) {
	return nil, errors.New("unexpected EditVideo call")
}

func (s *avatarContentServiceStub) DeleteContent(context.Context, *contentservice.DeleteContentReq, ...grpc.CallOption) (*contentservice.DeleteContentRes, error) {
	return nil, errors.New("unexpected DeleteContent call")
}

func TestUploadAvatarUploadsToObjectStorageAndReturnsURL(t *testing.T) {
	payload := makePNGFixture()
	uploaded := false

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://upload.example.com" {
			t.Fatalf("unexpected upload url: %s", r.URL.String())
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("multipart reader: %v", err)
		}
		fields := map[string]string{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("next multipart part: %v", err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("read multipart part: %v", err)
			}
			if part.FormName() == "file" {
				if !bytes.Equal(data, payload) {
					t.Fatalf("uploaded payload mismatch")
				}
				continue
			}
			fields[part.FormName()] = string(data)
		}
		if fields["key"] != "avatar/2026/04/15/test.png" {
			t.Fatalf("key = %q, want %q", fields["key"], "avatar/2026/04/15/test.png")
		}
		uploaded = true
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Header:     make(http.Header),
		}, nil
	})
	defer func() {
		http.DefaultTransport = oldTransport
	}()

	req := newAvatarUploadRequest(t, "avatar.png", payload)
	logic := NewUploadAvatarLogic(context.Background(), &svc.ServiceContext{
		ContentRpc: &avatarContentServiceStub{
			getUploadCredentialsFunc: func(_ context.Context, in *contentservice.GetUploadCredentialsReq, _ ...grpc.CallOption) (*contentservice.GetUploadCredentialsRes, error) {
				if in.GetScene() != "avatar" || in.GetFileExt() != ".png" || in.GetFileName() != "avatar.png" {
					t.Fatalf("unexpected credentials request: %+v", in)
				}
				return &contentpb.GetUploadCredentialsRes{
					ObjectKey: "avatar/2026/04/15/test.png",
					Url:       "https://cdn.example.com/avatar/2026/04/15/test.png",
					FormData: &contentpb.OssFormData{
						Host:             "https://upload.example.com",
						Policy:           "policy",
						Signature:        "signature",
						SignatureVersion: "OSS4-HMAC-SHA256",
						Credential:       "credential",
						Date:             "20260415T000000Z",
						Key:              "avatar/2026/04/15/test.png",
					},
				}, nil
			},
		},
	})

	resp, err := logic.UploadAvatar(req)
	if err != nil {
		t.Fatalf("UploadAvatar returned error: %v", err)
	}
	if !uploaded {
		t.Fatal("expected avatar to be uploaded to object storage")
	}
	if resp.Url != "https://cdn.example.com/avatar/2026/04/15/test.png" || resp.ObjectKey != "avatar/2026/04/15/test.png" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Mime != "image/png" || resp.Size != int64(len(payload)) {
		t.Fatalf("unexpected asset metadata: %+v", resp)
	}
}

func TestUploadAvatarRejectsUnsupportedFile(t *testing.T) {
	req := newAvatarUploadRequest(t, "avatar.txt", []byte("not-an-image"))
	logic := NewUploadAvatarLogic(context.Background(), &svc.ServiceContext{
		ContentRpc: &avatarContentServiceStub{},
	})

	if _, err := logic.UploadAvatar(req); err == nil {
		t.Fatal("expected unsupported file error")
	}
}

func newAvatarUploadRequest(t *testing.T, fileName string, payload []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/users/avatar/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func makePNGFixture() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0x18, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
