// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"
)

type UnlikeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnlikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlikeLogic {
	return &UnlikeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnlikeLogic) Unlike(req *types.UnlikeReq) (resp *types.UnlikeRes, err error) {
	if req == nil || req.ContentId == nil || req.Scene == nil {
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

	_, err = l.svcCtx.LikeRpc.Unlike(l.ctx, &interactionpb.UnlikeReq{
		UserId:    userID,
		ContentId: *req.ContentId,
		Scene:     scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.UnlikeRes{}, nil
}
