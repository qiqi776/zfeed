package logic

import (
	"context"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"zfeed/app/rpc/user/internal/do"
	"zfeed/app/rpc/user/internal/repositories"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewUpdateProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProfileLogic {
	return &UpdateProfileLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *UpdateProfileLogic) UpdateProfile(in *user.UpdateProfileReq) (*user.UpdateProfileRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	patch := &do.UserProfilePatch{
		UpdatedBy: in.GetUserId(),
	}
	hasUpdate := false

	if in.Nickname != nil {
		nickname := strings.TrimSpace(in.GetNickname())
		if nickname == "" {
			return nil, errorx.NewBadRequest("昵称不能为空")
		}
		patch.Nickname = &nickname
		hasUpdate = true
	}

	if in.Avatar != nil {
		avatar := strings.TrimSpace(in.GetAvatar())
		if !isAcceptedProfileAsset(avatar) {
			return nil, errorx.NewBadRequest("头像地址错误")
		}
		patch.Avatar = &avatar
		hasUpdate = true
	}

	if in.Bio != nil {
		bio := strings.TrimSpace(in.GetBio())
		patch.Bio = &bio
		hasUpdate = true
	}

	if in.Gender != nil {
		gender := int32(in.GetGender())
		if gender < 0 || gender > 2 {
			return nil, errorx.NewBadRequest("性别参数错误")
		}
		patch.Gender = &gender
		hasUpdate = true
	}

	if in.Email != nil {
		email := strings.TrimSpace(in.GetEmail())
		if email == "" {
			return nil, errorx.NewBadRequest("邮箱不能为空")
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, errorx.NewBadRequest("邮箱格式错误")
		}
		patch.Email = &email
		hasUpdate = true
	}

	if in.Birthday != nil {
		if in.GetBirthday() <= 0 {
			return nil, errorx.NewBadRequest("生日参数错误")
		}
		birthday := time.Unix(in.GetBirthday(), 0)
		patch.Birthday = &birthday
		hasUpdate = true
	}

	if !hasUpdate {
		return nil, errorx.NewBadRequest("没有可更新的字段")
	}

	userDO, err := l.userRepo.UpdateProfile(in.GetUserId(), patch)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新资料失败"))
	}
	if userDO == nil {
		return nil, errorx.NewNotFound("用户不存在")
	}

	return &user.UpdateProfileRes{
		UserInfo: buildPrivateUserInfo(userDO),
	}, nil
}

func isAcceptedProfileAsset(raw string) bool {
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "/") {
		return true
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
