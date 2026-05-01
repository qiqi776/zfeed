// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/mobilex"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterRes, err error) {
	if req == nil || req.Mobile == nil || req.Password == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if !mobilex.IsValid(*req.Mobile) {
		return nil, errorx.NewBadRequest("参数错误")
	}

	mobile := mobilex.Normalize(*req.Mobile)
	if mobile == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}

	var nickname string
	if req.Nickname != nil {
		nickname = *req.Nickname
	}

	var bio *string
	if req.Bio != nil {
		bio = req.Bio
	}

	var avatar string
	if req.Avatar != nil {
		avatar = *req.Avatar
	}

	var gender user.Gender
	if req.Gender != nil {
		gender = user.Gender(*req.Gender)
	}

	var email string
	if req.Email != nil {
		email = *req.Email
	}

	var birthday int64
	if req.Birthday != nil {
		birthday = *req.Birthday
	}

	rpcResp, err := l.svcCtx.UserRpc.Register(l.ctx, &user.RegisterReq{
		Mobile:   mobile,
		Password: *req.Password,
		Nickname: nickname,
		Avatar:   avatar,
		Bio:      bio,
		Email:    email,
		Gender:   gender,
		Birthday: birthday,
	})
	if err != nil {
		return nil, err
	}

	return &types.RegisterRes{
		UserId:    rpcResp.GetUserId(),
		Token:     rpcResp.GetToken(),
		ExpiredAt: rpcResp.GetExpiredAt(),
	}, nil
}
