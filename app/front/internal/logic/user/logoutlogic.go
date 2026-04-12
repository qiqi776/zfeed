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
	"zfeed/pkg/utils"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout() (resp *types.LogoutRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	tokenVal := l.ctx.Value("token")
	token, ok := tokenVal.(string)
	if !ok || token == "" {
		return nil, errorx.NewUnauthorized("token缺失")
	}

	_, err = l.svcCtx.UserRpc.Logout(l.ctx, &user.LogoutReq{
		UserId: userID,
		Token:  token,
	})
	if err != nil {
		return nil, err
	}

	return &types.LogoutRes{}, nil
}
