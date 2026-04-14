// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"

	"github.com/google/uuid"
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

func (l *UploadAvatarLogic) UploadAvatar(r *http.Request) (resp *types.UploadAvatarRes, err error) {
	if r == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if strings.TrimSpace(l.svcCtx.Config.Oss.UploadDir) == "" {
		return nil, errorx.NewInternal("上传目录未配置")
	}

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return nil, errorx.NewBadRequest("头像文件错误")
	}

	file, header, err := readAvatarFormFile(r)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sniffBuf := make([]byte, 512)
	readSize, readErr := io.ReadFull(file, sniffBuf)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return nil, errorx.Wrap(l.ctx, readErr, errorx.NewMsg("保存头像失败"))
	}
	sniffBuf = sniffBuf[:readSize]

	mimeType, ext, ok := detectAvatarAsset(sniffBuf)
	if !ok {
		return nil, errorx.NewBadRequest("头像文件类型错误")
	}

	objectKey := buildAvatarObjectKey(ext, time.Now())
	targetPath := filepath.Join(strings.TrimSpace(l.svcCtx.Config.Oss.UploadDir), filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存头像失败"))
	}

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存头像失败"))
	}
	defer targetFile.Close()

	if len(sniffBuf) > 0 {
		if _, err := targetFile.Write(sniffBuf); err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存头像失败"))
		}
	}
	written, err := io.Copy(targetFile, file)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存头像失败"))
	}

	size := written + int64(len(sniffBuf))
	if header != nil && header.Size > size {
		size = header.Size
	}

	return &types.UploadAvatarRes{
		Url:       buildAvatarPublicURL(l.svcCtx.Config.Oss.PublicHost, objectKey),
		ObjectKey: filepath.ToSlash(objectKey),
		Mime:      mimeType,
		Size:      size,
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

func buildAvatarObjectKey(ext string, now time.Time) string {
	return filepath.ToSlash(filepath.Join("avatar", now.Format("2006/01/02"), uuid.NewString()+ext))
}

func buildAvatarPublicURL(publicHost, objectKey string) string {
	normalizedKey := filepath.ToSlash(objectKey)
	host := strings.TrimRight(strings.TrimSpace(publicHost), "/")
	if host == "" {
		return "/" + normalizedKey
	}
	return host + "/" + normalizedKey
}
