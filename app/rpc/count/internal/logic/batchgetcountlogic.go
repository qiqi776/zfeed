package logic

import (
	"context"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetCountLogic {
	return &BatchGetCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BatchGetCountLogic) BatchGetCount(in *count.BatchGetCountReq) (*count.BatchGetCountRes, error) {
	// todo: add your logic here and delete this line

	return &count.BatchGetCountRes{}, nil
}
