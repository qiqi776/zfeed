package user

import (
	"context"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/client/userservice"
	userpb "zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProfileLogic {
	return &UpdateProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProfileLogic) UpdateProfile(req *types.UpdateProfileReq) (*types.UpdateProfileRes, error) {
	if emptyUpdateProfilePayload(req) ||
		invalidUpdateProfileGender(req.Gender) ||
		invalidUpdateProfileEmail(req.Email) ||
		invalidUpdateProfileAvatar(req.Avatar) ||
		invalidUpdateProfileBirthday(req.Birthday) {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	rpcReq := &userservice.UpdateProfileReq{
		UserId: userID,
	}

	if req.Nickname != nil {
		rpcReq.Nickname = req.Nickname
	}
	if req.Bio != nil {
		rpcReq.Bio = req.Bio
	}
	if req.Avatar != nil {
		avatar := strings.TrimSpace(*req.Avatar)
		rpcReq.Avatar = &avatar
	}
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		rpcReq.Email = &email
	}
	if req.Gender != nil {
		gender := userpb.Gender(*req.Gender)
		rpcReq.Gender = &gender
	}
	if req.Birthday != nil {
		rpcReq.Birthday = req.Birthday
	}

	rpcResp, err := l.svcCtx.UserRpc.UpdateProfile(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}
	if rpcResp == nil || rpcResp.GetUserInfo() == nil {
		return nil, errorx.NewMsg("更新资料失败")
	}

	return &types.UpdateProfileRes{
		UserInfo: types.UserInfo{
			UserId:   rpcResp.GetUserInfo().GetUserId(),
			Mobile:   rpcResp.GetUserInfo().GetMobile(),
			Nickname: rpcResp.GetUserInfo().GetNickname(),
			Avatar:   rpcResp.GetUserInfo().GetAvatar(),
			Bio:      rpcResp.GetUserInfo().GetBio(),
			Gender:   int32(rpcResp.GetUserInfo().GetGender()),
			Status:   int32(rpcResp.GetUserInfo().GetStatus()),
			Email:    rpcResp.GetUserInfo().GetEmail(),
			Birthday: rpcResp.GetUserInfo().GetBirthday(),
		},
	}, nil
}

func emptyUpdateProfilePayload(req *types.UpdateProfileReq) bool {
	if req == nil {
		return true
	}
	return req.Nickname == nil &&
		req.Bio == nil &&
		req.Avatar == nil &&
		req.Email == nil &&
		req.Gender == nil &&
		req.Birthday == nil
}

func invalidUpdateProfileGender(gender *int32) bool {
	if gender == nil {
		return false
	}
	switch userpb.Gender(*gender) {
	case userpb.Gender_GENDER_UNKNOWN, userpb.Gender_GENDER_MALE, userpb.Gender_GENDER_FEMALE:
		return false
	default:
		return true
	}
}

func invalidUpdateProfileEmail(email *string) bool {
	if email == nil {
		return false
	}
	value := strings.TrimSpace(*email)
	if value == "" {
		return true
	}
	address, err := mail.ParseAddress(value)
	return err != nil || address.Address != value
}

func invalidUpdateProfileAvatar(avatar *string) bool {
	if avatar == nil {
		return false
	}
	value := strings.TrimSpace(*avatar)
	if value == "" {
		return true
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

func invalidUpdateProfileBirthday(birthday *int64) bool {
	if birthday == nil {
		return false
	}
	if *birthday <= 0 {
		return true
	}
	return time.Unix(*birthday, 0).After(time.Now())
}
