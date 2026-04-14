package user

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"zfeed/app/front/internal/config"
	"zfeed/app/front/internal/svc"
)

func TestUploadAvatarStoresFileAndReturnsURL(t *testing.T) {
	tmpDir := t.TempDir()
	payload := makePNGFixture()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/users/avatar/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	logic := NewUploadAvatarLogic(context.Background(), &svc.ServiceContext{
		Config: config.Config{
			Oss: config.OssConfig{
				UploadDir:  tmpDir,
				PublicHost: "https://cdn.example.com",
			},
		},
	})

	resp, err := logic.UploadAvatar(req)
	if err != nil {
		t.Fatalf("UploadAvatar returned error: %v", err)
	}
	if resp.Mime != "image/png" {
		t.Fatalf("mime = %q, want %q", resp.Mime, "image/png")
	}
	if resp.Size <= 0 {
		t.Fatalf("size = %d, want > 0", resp.Size)
	}
	if resp.Url == "" || resp.ObjectKey == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	targetPath := filepath.Join(tmpDir, filepath.FromSlash(resp.ObjectKey))
	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
}

func TestUploadAvatarRejectsUnsupportedFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "avatar.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("not-an-image")); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/users/avatar/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	logic := NewUploadAvatarLogic(context.Background(), &svc.ServiceContext{
		Config: config.Config{
			Oss: config.OssConfig{
				UploadDir: t.TempDir(),
			},
		},
	})

	if _, err := logic.UploadAvatar(req); err == nil {
		t.Fatal("expected unsupported file error")
	}
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
