package logic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"zfeed/app/rpc/user/internal/common/utils/session"
	"zfeed/app/rpc/user/internal/do"
	"zfeed/app/rpc/user/internal/repositories"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RegisterLogic) Register(in *user.RegisterReq) (*user.RegisterRes, error) {
	if in == nil || in.GetMobile() == "" || in.GetPassword() == "" || in.GetAvatar() == "" || in.GetEmail() == "" || in.GetBirthday() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	nickname := in.GetNickname()
	if nickname == "" {
		nickname = in.GetMobile()
	}

	exist, err := l.userRepo.GetByMobile(in.GetMobile())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if exist != nil {
		return nil, errorx.NewMsg("手机号已注册")
	}

	passwordSalt, err := l.newPasswordSalt()
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成密码盐失败"))
	}

	passwordHash, err := utils.HashPassword(in.GetPassword() + passwordSalt)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("密码加密失败"))
	}

	userID, err := l.userRepo.Create(&do.UserDO{
		Username:     in.GetMobile(),
		Nickname:     nickname,
		Avatar:       in.GetAvatar(),
		Bio:          in.GetBio(),
		Mobile:       in.GetMobile(),
		Email:        in.GetEmail(),
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		Gender:       int32(in.GetGender()),
		Birthday:     l.truncateToDate(time.Unix(in.GetBirthday(), 0)),
		Status:       int32(user.UserStatus_USER_STATUS_ACTIVE),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建用户失败"))
	}

	sessionTTL := session.GetSessionTTL(l.svcCtx.Config)
	token := session.NewSessionToken()
	if err = session.SaveSession(l.ctx, l.svcCtx.Redis, userID, token, sessionTTL); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存登录态失败"))
	}

	return &user.RegisterRes{
		UserId:    userID,
		Token:     token,
		ExpiredAt: time.Now().Add(sessionTTL).Unix(),
	}, nil
}

func (l *RegisterLogic) newPasswordSalt() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(buf), nil
}

func (l *RegisterLogic) truncateToDate(t time.Time) *time.Time {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return &day
}
