package content

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"zfeed/app/front/internal/config"
)

func TestContentUploadsCredentialsBuildsCredentialPayload(t *testing.T) {
	fixedNow := time.Date(2026, 4, 14, 8, 30, 0, 0, time.UTC)
	builder := uploadCredentialBuilder{
		cfg: config.OssConfig{
			Provider:        "aliyun-oss",
			Region:          "cn-hangzhou",
			BucketName:      "zfeed-dev",
			AccessKeyId:     "test-ak",
			AccessKeySecret: "test-sk",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
			UploadDir:       "zfeed",
			PublicHost:      "https://cdn.example.com",
		},
		now: func() time.Time { return fixedNow },
	}

	resp, err := builder.build(uploadSceneArticleCover, ".png", "cover.png", 2048)
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}
	if !strings.HasPrefix(resp.ObjectKey, "zfeed/article-cover/2026/04/14/") {
		t.Fatalf("unexpected object key: %s", resp.ObjectKey)
	}
	if resp.FormData.Host != "https://oss-cn-hangzhou.aliyuncs.com" {
		t.Fatalf("unexpected host: %s", resp.FormData.Host)
	}
	if !strings.HasPrefix(resp.Url, "https://cdn.example.com/zfeed/article-cover/2026/04/14/") {
		t.Fatalf("unexpected public url: %s", resp.Url)
	}
	if resp.FormData.SignatureVersion != uploadSignatureVersion {
		t.Fatalf("unexpected signature version: %s", resp.FormData.SignatureVersion)
	}
	if resp.FormData.Credential != "test-ak/20260414/cn-hangzhou/oss/aliyun_v4_request" {
		t.Fatalf("unexpected credential: %s", resp.FormData.Credential)
	}
	if resp.FormData.Date != "20260414T083000Z" {
		t.Fatalf("unexpected date: %s", resp.FormData.Date)
	}
	policyPayload, err := base64.StdEncoding.DecodeString(resp.FormData.Policy)
	if err != nil {
		t.Fatalf("policy should be valid base64: %v", err)
	}
	var policy uploadPolicyDocument
	if err := json.Unmarshal(policyPayload, &policy); err != nil {
		t.Fatalf("policy should be valid json: %v", err)
	}
	if len(policy.Conditions) == 0 {
		t.Fatal("policy conditions should not be empty")
	}
	foundSizeLimit := false
	for _, item := range policy.Conditions {
		values, ok := item.([]any)
		if !ok || len(values) != 3 {
			continue
		}
		if values[0] != "content-length-range" {
			continue
		}
		maxValue, ok := values[2].(float64)
		if !ok {
			t.Fatalf("content-length-range max type = %T, want float64", values[2])
		}
		if int64(maxValue) != uploadMaxCoverSize {
			t.Fatalf("content-length-range max = %d, want %d", int64(maxValue), uploadMaxCoverSize)
		}
		foundSizeLimit = true
	}
	if !foundSizeLimit {
		t.Fatal("content-length-range policy condition should exist")
	}
	if resp.FormData.Signature == "" {
		t.Fatal("signature should not be empty")
	}
}

func TestContentUploadsCredentialsRejectsInvalidExt(t *testing.T) {
	builder := uploadCredentialBuilder{
		cfg: config.OssConfig{
			AccessKeyId:     "test-ak",
			AccessKeySecret: "test-sk",
			Region:          "cn-hangzhou",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		},
		now: time.Now,
	}

	if _, err := builder.build(uploadSceneArticleCover, ".mp4", "cover.mp4", 2048); err == nil {
		t.Fatal("expected invalid ext error")
	}
}

func TestContentUploadsCredentialsRejectsOversizedFile(t *testing.T) {
	builder := uploadCredentialBuilder{
		cfg: config.OssConfig{
			AccessKeyId:     "test-ak",
			AccessKeySecret: "test-sk",
			Region:          "cn-hangzhou",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		},
		now: time.Now,
	}

	if _, err := builder.build(uploadSceneAvatar, ".png", "avatar.png", uploadMaxAvatarSize+1); err == nil {
		t.Fatal("expected oversized file error")
	}
}

func TestContentUploadsCredentialsRejectsFileNameExtMismatch(t *testing.T) {
	builder := uploadCredentialBuilder{
		cfg: config.OssConfig{
			AccessKeyId:     "test-ak",
			AccessKeySecret: "test-sk",
			Region:          "cn-hangzhou",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		},
		now: time.Now,
	}

	if _, err := builder.build(uploadSceneVideoSource, ".mp4", "clip.mov", 1024); err == nil {
		t.Fatal("expected file name mismatch error")
	}
}
