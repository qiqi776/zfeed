package likeservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchQueryLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryLikeInfoLogic {
	return &BatchQueryLikeInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BatchQueryLikeInfoLogic) BatchQueryLikeInfo(in *interaction.BatchQueryLikeInfoReq) (*interaction.BatchQueryLikeInfoRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.BatchQueryLikeInfoRes{}, nil
}
