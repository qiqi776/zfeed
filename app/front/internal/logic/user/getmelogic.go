// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"sync"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	followservice "zfeed/app/rpc/interaction/client/followservice"
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
		userResp   *user.GetMeRes
		userErr    error
		countResp  = defaultUserProfileCounts()
		countErr   error
		followResp *followservice.GetFollowSummaryRes
		followErr  error
		wg         sync.WaitGroup
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
		followResp, followErr = loadFollowSummary(l.ctx, l.svcCtx, userID, nil)
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
	if followErr != nil {
		l.Errorf("query me follow summary failed, user_id=%d, err=%v", userID, followErr)
		followResp = nil
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
		FolloweeCount:         resolveFolloweeCount(countResp, followResp),
		FollowerCount:         resolveFollowerCount(countResp, followResp),
		LikeReceivedCount:     countResp.GetLikeCount(),
		FavoriteReceivedCount: countResp.GetFavoriteCount(),
		ContentCount:          countResp.GetContentCount(),
	}, nil
}
