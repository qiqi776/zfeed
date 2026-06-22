// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"net/mail"
	"net/url"
	"strings"
	"time"

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
	if req == nil || req.Mobile == nil || req.Password == nil || strings.TrimSpace(*req.Password) == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if !mobilex.IsValid(*req.Mobile) {
		return nil, errorx.NewBadRequest("参数错误")
	}

	mobile := mobilex.Normalize(*req.Mobile)
	if mobile == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if invalidRegisterGender(req.Gender) ||
		invalidRegisterBirthday(req.Birthday) ||
		invalidRegisterEmail(req.Email) ||
		invalidRegisterAvatar(req.Avatar) {
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
		avatar = strings.TrimSpace(*req.Avatar)
	}

	var gender user.Gender
	if req.Gender != nil {
		gender = user.Gender(*req.Gender)
	}

	var email string
	if req.Email != nil {
		email = strings.TrimSpace(*req.Email)
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
	if rpcResp == nil {
		return nil, errorx.NewMsg("注册失败")
	}

	return &types.RegisterRes{
		UserId:    rpcResp.GetUserId(),
		Token:     rpcResp.GetToken(),
		ExpiredAt: rpcResp.GetExpiredAt(),
	}, nil
}

func invalidRegisterGender(gender *int32) bool {
	if gender == nil {
		return false
	}
	switch user.Gender(*gender) {
	case user.Gender_GENDER_UNKNOWN, user.Gender_GENDER_MALE, user.Gender_GENDER_FEMALE:
		return false
	default:
		return true
	}
}

func invalidRegisterBirthday(birthday *int64) bool {
	if birthday == nil || *birthday == 0 {
		return false
	}
	if *birthday < 0 {
		return true
	}
	return time.Unix(*birthday, 0).After(time.Now())
}

func invalidRegisterEmail(email *string) bool {
	if email == nil {
		return false
	}
	value := strings.TrimSpace(*email)
	if value == "" {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err != nil || address.Address != value
}

func invalidRegisterAvatar(avatar *string) bool {
	if avatar == nil {
		return false
	}
	value := strings.TrimSpace(*avatar)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return false
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return true
	}
	if parsed.Host == "" {
		return true
	}
	return parsed.Scheme != "http" && parsed.Scheme != "https"
}
