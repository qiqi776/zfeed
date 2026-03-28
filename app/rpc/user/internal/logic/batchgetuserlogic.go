package logic

import (
	"context"

	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetUserLogic {
	return &BatchGetUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BatchGetUserLogic) BatchGetUser(in *user.BatchGetUserReq) (*user.BatchGetUserRes, error) {
	// todo: add your logic here and delete this line

	return &user.BatchGetUserRes{}, nil
}
