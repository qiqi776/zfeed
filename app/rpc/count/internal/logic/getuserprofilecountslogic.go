package logic

import (
	"context"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserProfileCountsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserProfileCountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserProfileCountsLogic {
	return &GetUserProfileCountsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserProfileCountsLogic) GetUserProfileCounts(in *count.GetUserProfileCountsReq) (*count.GetUserProfileCountsRes, error) {
	// todo: add your logic here and delete this line

	return &count.GetUserProfileCountsRes{}, nil
}
