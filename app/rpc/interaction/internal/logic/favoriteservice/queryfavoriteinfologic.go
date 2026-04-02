package favoriteservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFavoriteInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryFavoriteInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteInfoLogic {
	return &QueryFavoriteInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryFavoriteInfoLogic) QueryFavoriteInfo(in *interaction.QueryFavoriteInfoReq) (*interaction.QueryFavoriteInfoRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.QueryFavoriteInfoRes{}, nil
}
