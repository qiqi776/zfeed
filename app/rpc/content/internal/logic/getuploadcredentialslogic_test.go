package logic

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"zfeed/app/rpc/content/internal/config"
)

func TestUploadCredentialBuilderBuildsCredentialPayload(t *testing.T) {
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
	if !strings.HasPrefix(resp.GetObjectKey(), "zfeed/article-cover/2026/04/14/") {
		t.Fatalf("unexpected object key: %s", resp.GetObjectKey())
	}
	if resp.GetFormData().GetHost() != "https://oss-cn-hangzhou.aliyuncs.com" {
		t.Fatalf("unexpected host: %s", resp.GetFormData().GetHost())
	}
	if !strings.HasPrefix(resp.GetUrl(), "https://cdn.example.com/zfeed/article-cover/2026/04/14/") {
		t.Fatalf("unexpected public url: %s", resp.GetUrl())
	}
	if resp.GetFormData().GetSignatureVersion() != uploadSignatureVersion {
		t.Fatalf("unexpected signature version: %s", resp.GetFormData().GetSignatureVersion())
	}
	if resp.GetFormData().GetCredential() != "test-ak/20260414/cn-hangzhou/oss/aliyun_v4_request" {
		t.Fatalf("unexpected credential: %s", resp.GetFormData().GetCredential())
	}
	if resp.GetFormData().GetDate() != "20260414T083000Z" {
		t.Fatalf("unexpected date: %s", resp.GetFormData().GetDate())
	}

	policyPayload, err := base64.StdEncoding.DecodeString(resp.GetFormData().GetPolicy())
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
	if resp.GetFormData().GetSignature() == "" {
		t.Fatal("signature should not be empty")
	}
}

func TestUploadCredentialBuilderBuildsArticleImagePayload(t *testing.T) {
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

	resp, err := builder.build(uploadSceneArticleImage, ".webp", "inline-image.webp", 2048)
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}
	if !strings.HasPrefix(resp.GetObjectKey(), "zfeed/article-image/2026/04/14/") {
		t.Fatalf("unexpected object key: %s", resp.GetObjectKey())
	}
	if !strings.HasPrefix(resp.GetUrl(), "https://cdn.example.com/zfeed/article-image/2026/04/14/") {
		t.Fatalf("unexpected public url: %s", resp.GetUrl())
	}
}

func TestUploadCredentialBuilderRejectsInvalidExt(t *testing.T) {
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

func TestUploadCredentialBuilderRejectsOversizedFile(t *testing.T) {
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

func TestUploadCredentialBuilderRejectsFileNameExtMismatch(t *testing.T) {
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
