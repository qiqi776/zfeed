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

type UnFollowUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnFollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnFollowUserLogic {
	return &UnFollowUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnFollowUserLogic) UnFollowUser(req *types.UnFollowUserReq) (resp *types.UnFollowUserRes, err error) {
	if req == nil || req.TargetUserId == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	rpcResp, err := l.svcCtx.FollowRpc.UnfollowUser(l.ctx, &interaction.UnfollowUserReq{
		UserId:       userID,
		FollowUserId: *req.TargetUserId,
	})
	if err != nil {
		return nil, err
	}

	return &types.UnFollowUserRes{IsFollowed: rpcResp.GetIsFollowed()}, nil
}
