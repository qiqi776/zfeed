package content

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentUploadsCredentialsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContentUploadsCredentialsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentUploadsCredentialsLogic {
	return &ContentUploadsCredentialsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContentUploadsCredentialsLogic) ContentUploadsCredentials(req *types.ContentUploadsCredentialsReq) (*types.ContentUploadsCredentialsRes, error) {
	if req == nil || req.Scene == nil || req.FileExt == nil || req.FileSize == nil || req.FileName == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	rpcResp, err := l.svcCtx.ContentRpc.GetUploadCredentials(l.ctx, &contentservice.GetUploadCredentialsReq{
		Scene:    *req.Scene,
		FileExt:  *req.FileExt,
		FileSize: *req.FileSize,
		FileName: *req.FileName,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp.GetFormData() == nil {
		return nil, errorx.NewMsg("生成上传凭证失败")
	}

	return &types.ContentUploadsCredentialsRes{
		ObjectKey: rpcResp.GetObjectKey(),
		Url:       rpcResp.GetUrl(),
		ExpiredAt: rpcResp.GetExpiredAt(),
		FormData: types.OssFormData{
			Host:             rpcResp.GetFormData().GetHost(),
			Policy:           rpcResp.GetFormData().GetPolicy(),
			Signature:        rpcResp.GetFormData().GetSignature(),
			SecurityToken:    rpcResp.GetFormData().GetSecurityToken(),
			SignatureVersion: rpcResp.GetFormData().GetSignatureVersion(),
			Credential:       rpcResp.GetFormData().GetCredential(),
			Date:             rpcResp.GetFormData().GetDate(),
			Key:              rpcResp.GetFormData().GetKey(),
		},
	}, nil
}
