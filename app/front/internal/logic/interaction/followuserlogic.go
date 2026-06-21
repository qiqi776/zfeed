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

type FollowUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowUserLogic {
	return &FollowUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FollowUserLogic) FollowUser(req *types.FollowUserReq) (resp *types.FollowUserRes, err error) {
	if req == nil || req.TargetUserId == nil || *req.TargetUserId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}
	if *req.TargetUserId == userID {
		return nil, errorx.NewBadRequest("不能关注自己")
	}

	rpcResp, err := l.svcCtx.FollowRpc.FollowUser(l.ctx, &interaction.FollowUserReq{
		UserId:       userID,
		FollowUserId: *req.TargetUserId,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("关注用户失败")
	}

	return &types.FollowUserRes{IsFollowed: rpcResp.GetIsFollowed()}, nil
}
