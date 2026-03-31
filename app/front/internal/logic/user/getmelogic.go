// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeLogic {
	return &GetMeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMeLogic) GetMe() (resp *types.GetMeRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	rpcResp, err := l.svcCtx.UserRpc.GetMe(l.ctx, &user.GetMeReq{UserId: userID})
	if err != nil {
		return nil, err
	}
	if rpcResp.GetUserInfo() == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	return &types.GetMeRes{
		UserInfo: types.UserInfo{
			UserId:   rpcResp.GetUserInfo().GetUserId(),
			Mobile:   rpcResp.GetUserInfo().GetMobile(),
			Nickname: rpcResp.GetUserInfo().GetNickname(),
			Avatar:   rpcResp.GetUserInfo().GetAvatar(),
			Bio:      rpcResp.GetUserInfo().GetBio(),
			Gender:   int32(rpcResp.GetUserInfo().GetGender()),
			Status:   int32(rpcResp.GetUserInfo().GetStatus()),
		},
		FolloweeCount:         rpcResp.GetFolloweeCount(),
		FollowerCount:         rpcResp.GetFollowerCount(),
		LikeReceivedCount:     rpcResp.GetLikeReceivedCount(),
		FavoriteReceivedCount: rpcResp.GetFavoriteReceivedCount(),
	}, nil
}
