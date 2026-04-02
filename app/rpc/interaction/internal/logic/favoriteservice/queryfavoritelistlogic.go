package favoriteservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFavoriteListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryFavoriteListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteListLogic {
	return &QueryFavoriteListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryFavoriteListLogic) QueryFavoriteList(in *interaction.QueryFavoriteListReq) (*interaction.QueryFavoriteListRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.QueryFavoriteListRes{}, nil
}
