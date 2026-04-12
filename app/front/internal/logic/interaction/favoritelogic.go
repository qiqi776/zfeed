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

type FavoriteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FavoriteLogic {
	return &FavoriteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FavoriteLogic) Favorite(req *types.FavoriteReq) (resp *types.FavoriteRes, err error) {
	if req == nil || req.ContentId == nil || req.ContentUserId == nil || req.Scene == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	scene, err := parseScene(*req.Scene)
	if err != nil {
		return nil, err
	}

	_, err = l.svcCtx.FavoriteRpc.Favorite(l.ctx, &interaction.FavoriteReq{
		UserId:        userID,
		ContentId:     *req.ContentId,
		ContentUserId: *req.ContentUserId,
		Scene:         scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.FavoriteRes{}, nil
}
