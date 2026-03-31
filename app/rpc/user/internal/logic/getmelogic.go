package logic

import (
	"context"

	"zfeed/app/rpc/user/internal/repositories"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewGetMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeLogic {
	return &GetMeLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetMeLogic) GetMe(in *user.GetMeReq) (*user.GetMeRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	userDO, err := l.userRepo.GetByID(in.GetUserId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询我的信息失败"))
	}
	if userDO == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	return &user.GetMeRes{
		UserInfo: &user.UserInfo{
			UserId:   userDO.ID,
			Username: userDO.Username,
			Mobile:   userDO.Mobile,
			Nickname: userDO.Nickname,
			Avatar:   userDO.Avatar,
			Bio:      userDO.Bio,
			Gender:   user.Gender(userDO.Gender),
			Status:   user.UserStatus(userDO.Status),
		},
	}, nil
}
