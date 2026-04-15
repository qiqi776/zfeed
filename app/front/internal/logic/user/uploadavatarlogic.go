package user

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	contentservice "zfeed/app/rpc/content/contentservice"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadAvatarLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadAvatarLogic {
	return &UploadAvatarLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadAvatarLogic) UploadAvatar(r *http.Request) (*types.UploadAvatarRes, error) {
	if r == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return nil, errorx.NewBadRequest("头像文件错误")
	}

	file, header, err := readAvatarFormFile(r)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	payload, mimeType, ext, err := readAvatarPayload(file)
	if err != nil {
		return nil, err
	}

	fileName := "avatar" + ext
	if header != nil && strings.TrimSpace(header.Filename) != "" {
		fileName = header.Filename
	}

	credResp, err := l.svcCtx.ContentRpc.GetUploadCredentials(l.ctx, &contentservice.GetUploadCredentialsReq{
		Scene:    "avatar",
		FileExt:  ext,
		FileSize: int64(len(payload)),
		FileName: fileName,
	})
	if err != nil {
		return nil, err
	}
	if credResp.GetFormData() == nil {
		return nil, errorx.NewMsg("上传头像失败")
	}

	if err := uploadAvatarToObjectStorage(l.ctx, credResp.GetFormData(), fileName, mimeType, payload); err != nil {
		return nil, err
	}

	return &types.UploadAvatarRes{
		Url:       credResp.GetUrl(),
		ObjectKey: credResp.GetObjectKey(),
		Mime:      mimeType,
		Size:      int64(len(payload)),
	}, nil
}

func readAvatarFormFile(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	for _, field := range []string{"file", "avatar"} {
		file, header, err := r.FormFile(field)
		if err == nil {
			return file, header, nil
		}
	}
	return nil, nil, errorx.NewBadRequest("头像文件错误")
}

func readAvatarPayload(file multipart.File) ([]byte, string, string, error) {
	payload, err := io.ReadAll(io.LimitReader(file, 5<<20+1))
	if err != nil {
		return nil, "", "", errorx.NewMsg("上传头像失败")
	}
	if len(payload) == 0 {
		return nil, "", "", errorx.NewBadRequest("头像文件错误")
	}
	if len(payload) > 5<<20 {
		return nil, "", "", errorx.NewBadRequest("头像文件过大")
	}

	sniffLen := len(payload)
	if sniffLen > 512 {
		sniffLen = 512
	}
	mimeType, ext, ok := detectAvatarAsset(payload[:sniffLen])
	if !ok {
		return nil, "", "", errorx.NewBadRequest("头像文件类型错误")
	}
	return payload, mimeType, ext, nil
}

func detectAvatarAsset(sniff []byte) (mimeType string, ext string, ok bool) {
	contentType := http.DetectContentType(sniff)
	switch contentType {
	case "image/jpeg":
		return contentType, ".jpg", true
	case "image/png":
		return contentType, ".png", true
	case "image/webp":
		return contentType, ".webp", true
	default:
		return "", "", false
	}
}

func uploadAvatarToObjectStorage(ctx context.Context, form *contentservice.OssFormData, fileName, mimeType string, payload []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "policy", value: form.GetPolicy()},
		{name: "signature", value: form.GetSignature()},
		{name: "x-oss-security-token", value: form.GetSecurityToken()},
		{name: "x-oss-signature-version", value: form.GetSignatureVersion()},
		{name: "x-oss-credential", value: form.GetCredential()},
		{name: "x-oss-date", value: form.GetDate()},
		{name: "key", value: form.GetKey()},
	} {
		if field.value == "" {
			continue
		}
		if err := writer.WriteField(field.name, field.value); err != nil {
			return errorx.NewMsg("上传头像失败")
		}
	}

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return errorx.NewMsg("上传头像失败")
	}
	if _, err := part.Write(payload); err != nil {
		return errorx.NewMsg("上传头像失败")
	}
	if err := writer.Close(); err != nil {
		return errorx.NewMsg("上传头像失败")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, form.GetHost(), &body)
	if err != nil {
		return errorx.NewMsg("上传头像失败")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if mimeType != "" {
		req.Header.Set("X-Upload-Content-Type", mimeType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errorx.Wrap(ctx, err, errorx.NewMsg("上传头像失败"))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return errorx.NewMsg("上传头像失败")
}
