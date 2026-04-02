package logic

import (
	"context"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DecLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecLogic {
	return &DecLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DecLogic) Dec(in *count.DecReq) (*count.DecRes, error) {
	// todo: add your logic here and delete this line

	return &count.DecRes{}, nil
}
