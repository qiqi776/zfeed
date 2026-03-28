// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFavoriteInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryFavoriteInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteInfoLogic {
	return &QueryFavoriteInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryFavoriteInfoLogic) QueryFavoriteInfo(req *types.QueryFavoriteInfoReq) (resp *types.QueryFavoriteInfoRes, err error) {
	// todo: add your logic here and delete this line

	return
}
