package contentservicelogic

import (
	"context"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishVideoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishVideoLogic {
	return &PublishVideoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PublishVideoLogic) PublishVideo(in *content.VideoPublishReq) (*content.VideoPublishRes, error) {
	// todo: add your logic here and delete this line

	return &content.VideoPublishRes{}, nil
}
