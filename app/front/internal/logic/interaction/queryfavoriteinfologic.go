// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/interaction/interaction"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

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
	if req == nil || req.ContentId == nil || req.Scene == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	scene, err := parseScene(*req.Scene)
	if err != nil {
		return nil, err
	}

	userID := utils.GetContextUserIdWithDefault(l.ctx)
	rpcResp, err := l.svcCtx.FavoriteRpc.QueryFavoriteInfo(l.ctx, &interaction.QueryFavoriteInfoReq{
		UserId:    userID,
		ContentId: *req.ContentId,
		Scene:     scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryFavoriteInfoRes{
		FavoriteCount: rpcResp.GetFavoriteCount(),
		IsFavorite:    rpcResp.GetIsFavorited(),
		ContentId:     rpcResp.GetContentId(),
		Scene:         rpcResp.GetScene().String(),
	}, nil
}
