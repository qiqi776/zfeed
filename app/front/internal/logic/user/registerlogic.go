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
	if req == nil || req.Mobile == nil || req.Password == nil || req.Avatar == nil || req.Gender == nil || req.Email == nil || req.Birthday == nil {
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

	rpcResp, err := l.svcCtx.UserRpc.Register(l.ctx, &user.RegisterReq{
		Mobile:   *req.Mobile,
		Password: *req.Password,
		Nickname: nickname,
		Avatar:   *req.Avatar,
		Bio:      bio,
		Email:    *req.Email,
		Gender:   user.Gender(*req.Gender),
		Birthday: *req.Birthday,
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
