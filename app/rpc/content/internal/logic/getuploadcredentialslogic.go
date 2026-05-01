package logic

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"path"
	"strings"
	"time"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUploadCredentialsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUploadCredentialsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUploadCredentialsLogic {
	return &GetUploadCredentialsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUploadCredentialsLogic) GetUploadCredentials(in *content.GetUploadCredentialsReq) (*content.GetUploadCredentialsRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	builder := uploadCredentialBuilder{
		cfg: l.svcCtx.Config.Oss,
		now: time.Now,
	}
	return builder.build(in.GetScene(), in.GetFileExt(), in.GetFileName(), in.GetFileSize())
}

const (
	uploadSceneAvatar       = "avatar"
	uploadSceneArticleCover = "article-cover"
	uploadSceneArticleImage = "article-image"
	uploadSceneVideoCover   = "video-cover"
	uploadSceneVideoSource  = "video-source"

	uploadSignatureVersion = "OSS4-HMAC-SHA256"
	uploadServiceName      = "oss"
	uploadRequestSuffix    = "aliyun_v4_request"
	uploadPolicyTTL        = 15 * time.Minute

	uploadMaxAvatarSize       = 5 << 20
	uploadMaxCoverSize        = 10 << 20
	uploadMaxVideoSourceSize  = 512 << 20
	uploadMaxOriginalFileName = 255
)

type uploadCredentialBuilder struct {
	cfg config.OssConfig
	now func() time.Time
}

type uploadPolicyDocument struct {
	Expiration string `json:"expiration"`
	Conditions []any  `json:"conditions"`
}

type uploadScenePolicy struct {
	maxSize    int64
	allowedExt map[string]struct{}
}

var uploadScenePolicies = map[string]uploadScenePolicy{
	uploadSceneAvatar: {
		maxSize: uploadMaxAvatarSize,
		allowedExt: map[string]struct{}{
			".jpg":  {},
			".jpeg": {},
			".png":  {},
			".webp": {},
		},
	},
	uploadSceneArticleCover: {
		maxSize: uploadMaxCoverSize,
		allowedExt: map[string]struct{}{
			".jpg":  {},
			".jpeg": {},
			".png":  {},
			".webp": {},
		},
	},
	uploadSceneArticleImage: {
		maxSize: uploadMaxCoverSize,
		allowedExt: map[string]struct{}{
			".jpg":  {},
			".jpeg": {},
			".png":  {},
			".webp": {},
		},
	},
	uploadSceneVideoCover: {
		maxSize: uploadMaxCoverSize,
		allowedExt: map[string]struct{}{
			".jpg":  {},
			".jpeg": {},
			".png":  {},
			".webp": {},
		},
	},
	uploadSceneVideoSource: {
		maxSize: uploadMaxVideoSourceSize,
		allowedExt: map[string]struct{}{
			".mp4":  {},
			".mov":  {},
			".m4v":  {},
			".webm": {},
		},
	},
}

func (b uploadCredentialBuilder) build(scene, fileExt, fileName string, fileSize int64) (*content.GetUploadCredentialsRes, error) {
	scene = strings.TrimSpace(scene)
	fileExt = normalizeFileExt(fileExt)
	fileName = strings.TrimSpace(fileName)

	scenePolicy, ok := uploadScenePolicies[scene]
	if !ok {
		return nil, errorx.NewBadRequest("上传场景错误")
	}
	if !scenePolicy.allowsExt(fileExt) {
		return nil, errorx.NewBadRequest("文件类型错误")
	}
	if fileName == "" || fileSize <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if !isValidUploadFileName(fileName, fileExt) {
		return nil, errorx.NewBadRequest("文件名错误")
	}
	if fileSize > scenePolicy.maxSize {
		return nil, errorx.NewBadRequest("文件过大")
	}
	if strings.TrimSpace(b.cfg.AccessKeyId) == "" || strings.TrimSpace(b.cfg.AccessKeySecret) == "" ||
		strings.TrimSpace(b.cfg.Region) == "" || strings.TrimSpace(b.cfg.Endpoint) == "" {
		return nil, errorx.NewInternal("上传配置未就绪")
	}

	now := b.now().UTC()
	shortDate := now.Format("20060102")
	dateTime := now.Format("20060102T150405Z")
	objectKey := buildObjectKey(b.cfg.UploadDir, scene, fileExt, now)
	credential := strings.TrimSpace(b.cfg.AccessKeyId) + "/" + shortDate + "/" + strings.TrimSpace(b.cfg.Region) + "/" + uploadServiceName + "/" + uploadRequestSuffix
	expiredAt := now.Add(uploadPolicyTTL)

	policy := uploadPolicyDocument{
		Expiration: expiredAt.Format(time.RFC3339),
		Conditions: []any{
			map[string]string{"x-oss-signature-version": uploadSignatureVersion},
			map[string]string{"x-oss-credential": credential},
			map[string]string{"x-oss-date": dateTime},
			map[string]string{"key": objectKey},
			[]any{"content-length-range", 1, scenePolicy.maxSize},
		},
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return nil, errorx.Wrap(context.Background(), err, errorx.NewMsg("生成上传凭证失败"))
	}
	policyBase64 := base64.StdEncoding.EncodeToString(policyJSON)

	signingKey := deriveSigningKey(strings.TrimSpace(b.cfg.AccessKeySecret), shortDate, strings.TrimSpace(b.cfg.Region))
	signature := hex.EncodeToString(hmacSHA256(signingKey, policyBase64))

	return &content.GetUploadCredentialsRes{
		ObjectKey: objectKey,
		Url:       buildUploadPublicURL(b.cfg.PublicHost, normalizeUploadHost(b.cfg.Endpoint), objectKey),
		ExpiredAt: expiredAt.Unix(),
		FormData: &content.OssFormData{
			Host:             normalizeUploadHost(b.cfg.Endpoint),
			Policy:           policyBase64,
			Signature:        signature,
			SecurityToken:    "",
			SignatureVersion: uploadSignatureVersion,
			Credential:       credential,
			Date:             dateTime,
			Key:              objectKey,
		},
	}, nil
}

func (p uploadScenePolicy) allowsExt(fileExt string) bool {
	_, ok := p.allowedExt[fileExt]
	return ok
}

func isValidUploadFileName(fileName, fileExt string) bool {
	if fileName == "" || len(fileName) > uploadMaxOriginalFileName {
		return false
	}
	if strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return false
	}
	base := path.Base(fileName)
	if base == "." || base == ".." || strings.TrimSpace(base) == "" {
		return false
	}
	return normalizeFileExt(path.Ext(base)) == fileExt
}

func normalizeFileExt(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, ".") {
		return value
	}
	return "." + value
}

func buildObjectKey(uploadDir, scene, fileExt string, now time.Time) string {
	prefix := strings.Trim(strings.TrimSpace(uploadDir), "/")
	if prefix == "" {
		prefix = "uploads"
	}
	return path.Join(prefix, scene, now.Format("2006/01/02"), uuid.NewString()+fileExt)
}

func normalizeUploadHost(endpoint string) string {
	value := strings.TrimSpace(endpoint)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return "https://" + value
}

func buildUploadPublicURL(publicHost, uploadHost, objectKey string) string {
	host := strings.TrimRight(strings.TrimSpace(publicHost), "/")
	if host == "" {
		host = strings.TrimRight(strings.TrimSpace(uploadHost), "/")
	}
	if host == "" {
		return "/" + objectKey
	}
	return host + "/" + objectKey
}

func deriveSigningKey(secret, shortDate, region string) []byte {
	dateKey := hmacSHA256([]byte("aliyun_v4"+secret), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, uploadServiceName)
	return hmacSHA256(serviceKey, uploadRequestSuffix)
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}
