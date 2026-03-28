package logic

import (
	"context"

	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeLogic {
	return &GetMeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMeLogic) GetMe(in *user.GetMeReq) (*user.GetMeRes, error) {
	// todo: add your logic here and delete this line

	return &user.GetMeRes{}, nil
}
