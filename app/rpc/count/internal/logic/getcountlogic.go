package logic

import (
	"context"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCountLogic {
	return &GetCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCountLogic) GetCount(in *count.GetCountReq) (*count.GetCountRes, error) {
	// todo: add your logic here and delete this line

	return &count.GetCountRes{}, nil
}
