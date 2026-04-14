package user

import (
	"context"
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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

type updateProfileRow struct {
	ID       int64      `gorm:"column:id"`
	Mobile   string     `gorm:"column:mobile"`
	Nickname string     `gorm:"column:nickname"`
	Avatar   string     `gorm:"column:avatar"`
	Bio      string     `gorm:"column:bio"`
	Gender   int32      `gorm:"column:gender"`
	Status   int32      `gorm:"column:status"`
	Birthday *time.Time `gorm:"column:birthday"`
	Email    string     `gorm:"column:email"`
}

func (l *UpdateProfileLogic) UpdateProfile(req *types.UpdateProfileReq) (*types.UpdateProfileRes, error) {
	if req == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	updates := map[string]any{
		"updated_by": userID,
	}
	hasUpdate := false

	if req.Nickname != nil {
		nickname := strings.TrimSpace(*req.Nickname)
		if nickname == "" {
			return nil, errorx.NewBadRequest("昵称不能为空")
		}
		updates["nickname"] = nickname
		hasUpdate = true
	}

	if req.Avatar != nil {
		avatar := strings.TrimSpace(*req.Avatar)
		if !isAcceptedProfileAsset(avatar) {
			return nil, errorx.NewBadRequest("头像地址错误")
		}
		updates["avatar"] = avatar
		hasUpdate = true
	}

	if req.Bio != nil {
		updates["bio"] = strings.TrimSpace(*req.Bio)
		hasUpdate = true
	}

	if req.Gender != nil {
		if *req.Gender < 0 || *req.Gender > 2 {
			return nil, errorx.NewBadRequest("性别参数错误")
		}
		updates["gender"] = *req.Gender
		hasUpdate = true
	}

	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if email == "" {
			return nil, errorx.NewBadRequest("邮箱不能为空")
		}
		if _, parseErr := mail.ParseAddress(email); parseErr != nil {
			return nil, errorx.NewBadRequest("邮箱格式错误")
		}
		updates["email"] = email
		hasUpdate = true
	}

	if req.Birthday != nil {
		if *req.Birthday <= 0 {
			return nil, errorx.NewBadRequest("生日参数错误")
		}
		birthday := time.Unix(*req.Birthday, 0)
		updates["birthday"] = &birthday
		hasUpdate = true
	}

	if !hasUpdate {
		return nil, errorx.NewBadRequest("没有可更新的字段")
	}

	result := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_user").
		Where("id = ? AND is_deleted = 0", userID).
		Updates(updates)
	if result.Error != nil {
		return nil, errorx.Wrap(l.ctx, result.Error, errorx.NewMsg("更新资料失败"))
	}
	if result.RowsAffected == 0 {
		return nil, errorx.NewNotFound("用户不存在")
	}

	row, err := l.queryUpdatedUser(userID)
	if err != nil {
		return nil, err
	}

	return &types.UpdateProfileRes{
		UserInfo: types.UserInfo{
			UserId:   row.ID,
			Mobile:   row.Mobile,
			Nickname: row.Nickname,
			Avatar:   row.Avatar,
			Bio:      row.Bio,
			Gender:   row.Gender,
			Status:   row.Status,
			Email:    row.Email,
			Birthday: unixOrZero(row.Birthday),
		},
	}, nil
}

func (l *UpdateProfileLogic) queryUpdatedUser(userID int64) (*updateProfileRow, error) {
	var row updateProfileRow
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_user").
		Select("id", "mobile", "nickname", "avatar", "bio", "gender", "status", "birthday", "email").
		Where("id = ? AND is_deleted = 0", userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewNotFound("用户不存在")
		}
		return nil, err
	}
	return &row, nil
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
