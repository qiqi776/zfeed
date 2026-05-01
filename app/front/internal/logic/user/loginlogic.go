// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/mobilex"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginRes, err error) {
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

	rpcResp, err := l.svcCtx.UserRpc.Login(l.ctx, &user.LoginReq{
		Mobile:   mobile,
		Password: *req.Password,
	})
	if err != nil {
		return nil, err
	}

	return &types.LoginRes{
		UserId:    rpcResp.UserId,
		Token:     rpcResp.Token,
		ExpiredAt: rpcResp.ExpiredAt,
		Nickname:  rpcResp.Nickname,
		Avatar:    rpcResp.Avatar,
	}, nil
}
