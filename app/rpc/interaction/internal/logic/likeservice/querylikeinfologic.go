package likeservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLikeInfoLogic {
	return &QueryLikeInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryLikeInfoLogic) QueryLikeInfo(in *interaction.QueryLikeInfoReq) (*interaction.QueryLikeInfoRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.QueryLikeInfoRes{}, nil
}
