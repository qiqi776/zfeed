package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/user/internal/common/utils/session"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LogoutLogic) Logout(in *user.LogoutReq) (*user.LogoutRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	if err := session.RemoveSession(l.ctx, l.svcCtx.Redis, in.GetUserId(), in.GetToken()); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("退出登录失败"))
	}

	return &user.LogoutRes{}, nil
}
