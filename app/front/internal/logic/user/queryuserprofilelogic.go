// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryUserProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryUserProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserProfileLogic {
	return &QueryUserProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserProfileLogic) QueryUserProfile(req *types.QueryUserProfileReq) (resp *types.QueryUserProfileRes, err error) {
	if req == nil || req.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	rpcResp, err := l.svcCtx.UserRpc.GetUserProfile(l.ctx, &user.GetUserProfileReq{
		UserId: req.UserId,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp.GetUserProfile() == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	// Profile counts and viewer relation are placeholders until interaction/count services exist.
	return &types.QueryUserProfileRes{
		UserProfileInfo: types.UserProfileInfo{
			UserId:   rpcResp.GetUserProfile().GetUserId(),
			Nickname: rpcResp.GetUserProfile().GetNickname(),
			Avatar:   rpcResp.GetUserProfile().GetAvatar(),
			Bio:      rpcResp.GetUserProfile().GetBio(),
			Gender:   int32(rpcResp.GetUserProfile().GetGender()),
		},
		UserProfileCounts: types.UserProfileCounts{},
		ViewerProfileState: types.ViewerProfileState{
			IsFollowing: false,
		},
	}, nil
}
