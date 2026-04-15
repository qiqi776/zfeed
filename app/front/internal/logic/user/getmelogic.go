// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"sync"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
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
		wg        sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		userResp, userErr = l.svcCtx.UserRpc.GetMe(l.ctx, &user.GetMeReq{UserId: userID})
	}()

	go func() {
		defer wg.Done()
		countResp, countErr = loadUserProfileCounts(l.ctx, l.svcCtx, userID)
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

	return &types.GetMeRes{
		UserInfo: types.UserInfo{
			UserId:   userResp.GetUserInfo().GetUserId(),
			Mobile:   userResp.GetUserInfo().GetMobile(),
			Nickname: userResp.GetUserInfo().GetNickname(),
			Avatar:   userResp.GetUserInfo().GetAvatar(),
			Bio:      userResp.GetUserInfo().GetBio(),
			Gender:   int32(userResp.GetUserInfo().GetGender()),
			Status:   int32(userResp.GetUserInfo().GetStatus()),
			Email:    userResp.GetUserInfo().GetEmail(),
			Birthday: userResp.GetUserInfo().GetBirthday(),
		},
		FolloweeCount:         countResp.GetFollowingCount(),
		FollowerCount:         countResp.GetFollowedCount(),
		LikeReceivedCount:     countResp.GetLikeCount(),
		FavoriteReceivedCount: countResp.GetFavoriteCount(),
		ContentCount:          countResp.GetContentCount(),
	}, nil
}
