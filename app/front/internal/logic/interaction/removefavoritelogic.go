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

type RemoveFavoriteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFavoriteLogic {
	return &RemoveFavoriteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveFavoriteLogic) RemoveFavorite(req *types.RemoveFavoriteReq) (resp *types.RemoveFavoriteRes, err error) {
	if req == nil || req.ContentId == nil || req.Scene == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	scene, err := parseScene(*req.Scene)
	if err != nil {
		return nil, err
	}

	_, err = l.svcCtx.FavoriteRpc.RemoveFavorite(l.ctx, &interaction.RemoveFavoriteReq{
		UserId:    userID,
		ContentId: *req.ContentId,
		Scene:     scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.RemoveFavoriteRes{}, nil
}
