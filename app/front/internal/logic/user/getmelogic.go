// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"errors"
	"sync"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetMeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeLogic {
	return &GetMeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMeLogic) GetMe() (resp *types.GetMeRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	var (
		userResp  *user.GetMeRes
		userErr   error
		countResp = defaultUserProfileCounts()
		countErr  error
		extraResp *meProfileExtra
		extraErr  error
		wg        sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		userResp, userErr = l.svcCtx.UserRpc.GetMe(l.ctx, &user.GetMeReq{UserId: userID})
	}()

	go func() {
		defer wg.Done()
		countResp, countErr = loadUserProfileCounts(l.ctx, l.svcCtx, userID)
	}()

	go func() {
		defer wg.Done()
		extraResp, extraErr = l.queryProfileExtra(userID)
	}()

	wg.Wait()

	if userErr != nil {
		return nil, userErr
	}
	if userResp.GetUserInfo() == nil {
		return nil, errorx.NewNotFound("用户不存在")
	}
	if countErr != nil {
		l.Errorf("query me counts failed, user_id=%d, err=%v", userID, countErr)
		countResp = defaultUserProfileCounts()
	}
	if countResp == nil {
		countResp = defaultUserProfileCounts()
	}
	if extraErr != nil {
		l.Errorf("query me profile extra failed, user_id=%d, err=%v", userID, extraErr)
		extraResp = &meProfileExtra{}
	}
	if extraResp == nil {
		extraResp = &meProfileExtra{}
	}

	return &types.GetMeRes{
		UserInfo: types.UserInfo{
			UserId:   userResp.GetUserInfo().GetUserId(),
			Mobile:   userResp.GetUserInfo().GetMobile(),
			Nickname: userResp.GetUserInfo().GetNickname(),
			Avatar:   userResp.GetUserInfo().GetAvatar(),
			Bio:      userResp.GetUserInfo().GetBio(),
			Gender:   int32(userResp.GetUserInfo().GetGender()),
			Status:   int32(userResp.GetUserInfo().GetStatus()),
			Email:    extraResp.Email,
			Birthday: unixOrZero(extraResp.Birthday),
		},
		FolloweeCount:         countResp.GetFollowingCount(),
		FollowerCount:         countResp.GetFollowedCount(),
		LikeReceivedCount:     countResp.GetLikeCount(),
		FavoriteReceivedCount: countResp.GetFavoriteCount(),
		ContentCount:          countResp.GetContentCount(),
	}, nil
}

type meProfileExtra struct {
	Email    string     `gorm:"column:email"`
	Birthday *time.Time `gorm:"column:birthday"`
}

func (l *GetMeLogic) queryProfileExtra(userID int64) (*meProfileExtra, error) {
	if userID <= 0 || l.svcCtx == nil || l.svcCtx.MysqlDb == nil {
		return &meProfileExtra{}, nil
	}

	var row meProfileExtra
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_user").
		Select("email", "birthday").
		Where("id = ? AND is_deleted = 0", userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &meProfileExtra{}, nil
		}
		return nil, err
	}
	return &row, nil
}

func unixOrZero(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}
