package logic

import (
	"context"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type IncLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IncLogic {
	return &IncLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *IncLogic) Inc(in *count.IncReq) (*count.IncRes, error) {
	// todo: add your logic here and delete this line

	return &count.IncRes{}, nil
}
