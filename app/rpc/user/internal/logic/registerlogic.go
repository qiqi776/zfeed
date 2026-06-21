package logic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/mail"
	"strings"
	"time"

	"zfeed/app/rpc/user/internal/common/utils/session"
	"zfeed/app/rpc/user/internal/do"
	"zfeed/app/rpc/user/internal/model"
	"zfeed/app/rpc/user/internal/repositories"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/mobilex"
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
	if in == nil || in.GetMobile() == "" || strings.TrimSpace(in.GetPassword()) == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if !mobilex.IsValid(in.GetMobile()) {
		return nil, errorx.NewBadRequest("参数错误")
	}

	mobile := mobilex.Normalize(in.GetMobile())
	if mobile == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}

	nickname := resolveRegisterNickname(mobile, in.GetNickname())
	avatar := strings.TrimSpace(in.GetAvatar())
	if avatar != "" && !isAcceptedProfileAsset(avatar) {
		return nil, errorx.NewBadRequest("头像地址错误")
	}
	bio := strings.TrimSpace(in.GetBio())
	email := resolveRegisterEmail(mobile, in.GetEmail())
	if strings.TrimSpace(in.GetEmail()) != "" {
		parsedEmail, err := mail.ParseAddress(email)
		if err != nil || parsedEmail.Address != email {
			return nil, errorx.NewBadRequest("邮箱格式错误")
		}
	}
	birthdayInput := in.GetBirthday()
	if birthdayInput < 0 {
		return nil, errorx.NewBadRequest("生日参数错误")
	}
	if birthdayInput > 0 && time.Unix(birthdayInput, 0).After(time.Now()) {
		return nil, errorx.NewBadRequest("生日参数错误")
	}
	birthday := resolveRegisterBirthday(birthdayInput)
	gender := int32(in.GetGender())
	if gender < 0 || gender > 2 {
		return nil, errorx.NewBadRequest("性别参数错误")
	}

	exist, err := l.userRepo.GetByMobile(mobile)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if exist != nil {
		return nil, errorx.NewConflict("手机号已注册")
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
		Username:     mobile,
		Nickname:     nickname,
		Avatar:       avatar,
		Bio:          bio,
		Mobile:       mobile,
		Email:        email,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		Gender:       gender,
		Birthday:     birthday,
		Status:       int32(user.UserStatus_USER_STATUS_ACTIVE),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建用户失败"))
	}

	sessionTTL := session.GetSessionTTL(l.svcCtx.Config)
	token := session.NewSessionToken()
	if err = session.SaveSession(l.ctx, l.svcCtx.Redis, userID, token, sessionTTL); err != nil {
		if deleteErr := l.svcCtx.MysqlDb.WithContext(l.ctx).Delete(&model.ZfeedUser{}, userID).Error; deleteErr != nil {
			logx.WithContext(l.ctx).Errorf("回滚注册用户失败: %v", deleteErr)
		}
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

func resolveRegisterNickname(mobile string, nickname string) string {
	trimmed := strings.TrimSpace(nickname)
	if trimmed != "" {
		return trimmed
	}
	return mobile
}

func resolveRegisterEmail(mobile string, email string) string {
	trimmed := strings.TrimSpace(email)
	if trimmed != "" {
		return trimmed
	}

	var digits strings.Builder
	for _, ch := range mobile {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}

	suffix := digits.String()
	if suffix == "" {
		suffix = "user"
	}

	return "register-" + suffix + "@zfeed.local"
}

func resolveRegisterBirthday(unix int64) *time.Time {
	if unix <= 0 {
		return nil
	}
	return truncateDate(time.Unix(unix, 0))
}

func truncateDate(t time.Time) *time.Time {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return &day
}
