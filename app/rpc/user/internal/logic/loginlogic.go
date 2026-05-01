package logic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/user/internal/common/utils/session"
	"zfeed/app/rpc/user/internal/repositories"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/mobilex"
	"zfeed/pkg/utils"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *LoginLogic) Login(in *user.LoginReq) (*user.LoginRes, error) {
	if in == nil || in.GetPassword() == "" || in.GetMobile() == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if !mobilex.IsValid(in.GetMobile()) {
		return nil, errorx.NewBadRequest("参数错误")
	}

	mobile := mobilex.Normalize(in.GetMobile())
	if mobile == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}

	u, err := l.userRepo.GetByMobile(mobile)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if u == nil {
		return nil, errorx.NewNotFound("用户不存在")
	}

	if !utils.CheckPassword(u.PasswordHash, in.GetPassword()+u.PasswordSalt) {
		return nil, errorx.NewUnauthorized("密码错误")
	}

	sessionTTL := session.GetSessionTTL(l.svcCtx.Config)
	token := session.NewSessionToken()
	if err := session.SaveSession(l.ctx, l.svcCtx.Redis, u.ID, token, sessionTTL); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存登录态失败"))
	}

	return &user.LoginRes{
		UserId:    u.ID,
		Token:     token,
		ExpiredAt: time.Now().Add(sessionTTL).Unix(),
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
	}, nil
}
