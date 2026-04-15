package user

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/client/userservice"
	userpb "zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProfileLogic {
	return &UpdateProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProfileLogic) UpdateProfile(req *types.UpdateProfileReq) (*types.UpdateProfileRes, error) {
	if req == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	rpcReq := &userservice.UpdateProfileReq{
		UserId: userID,
	}

	if req.Nickname != nil {
		rpcReq.Nickname = req.Nickname
	}
	if req.Bio != nil {
		rpcReq.Bio = req.Bio
	}
	if req.Avatar != nil {
		rpcReq.Avatar = req.Avatar
	}
	if req.Email != nil {
		rpcReq.Email = req.Email
	}
	if req.Gender != nil {
		gender := userpb.Gender(*req.Gender)
		rpcReq.Gender = &gender
	}
	if req.Birthday != nil {
		rpcReq.Birthday = req.Birthday
	}

	rpcResp, err := l.svcCtx.UserRpc.UpdateProfile(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}
	if rpcResp == nil || rpcResp.GetUserInfo() == nil {
		return nil, errorx.NewMsg("更新资料失败")
	}

	return &types.UpdateProfileRes{
		UserInfo: types.UserInfo{
			UserId:   rpcResp.GetUserInfo().GetUserId(),
			Mobile:   rpcResp.GetUserInfo().GetMobile(),
			Nickname: rpcResp.GetUserInfo().GetNickname(),
			Avatar:   rpcResp.GetUserInfo().GetAvatar(),
			Bio:      rpcResp.GetUserInfo().GetBio(),
			Gender:   int32(rpcResp.GetUserInfo().GetGender()),
			Status:   int32(rpcResp.GetUserInfo().GetStatus()),
			Email:    rpcResp.GetUserInfo().GetEmail(),
			Birthday: rpcResp.GetUserInfo().GetBirthday(),
		},
	}, nil
}
