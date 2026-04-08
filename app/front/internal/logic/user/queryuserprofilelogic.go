// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"sync"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryUserProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryUserProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserProfileLogic {
	return &QueryUserProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserProfileLogic) QueryUserProfile(req *types.QueryUserProfileReq) (resp *types.QueryUserProfileRes, err error) {
	if req == nil || req.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	var (
		userResp  *user.GetUserProfileRes
		userErr   error
		countResp *count.GetUserProfileCountsRes
		countErr  error
		wg        sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		userResp, userErr = l.svcCtx.UserRpc.GetUserProfile(l.ctx, &user.GetUserProfileReq{
			UserId: req.UserId,
		})
	}()

	go func() {
		defer wg.Done()
		countResp, countErr = loadUserProfileCounts(l.ctx, l.svcCtx, req.UserId)
	}()

	wg.Wait()

	if userErr != nil {
		return nil, userErr
	}
	if userResp.GetUserProfile() == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	if countErr != nil {
		l.Errorf("query user profile counts failed, user_id=%d, err=%v", req.UserId, countErr)
		countResp = defaultUserProfileCounts()
	}
	if countResp == nil {
		countResp = defaultUserProfileCounts()
	}

	return &types.QueryUserProfileRes{
		UserProfileInfo: types.UserProfileInfo{
			UserId:   userResp.GetUserProfile().GetUserId(),
			Nickname: userResp.GetUserProfile().GetNickname(),
			Avatar:   userResp.GetUserProfile().GetAvatar(),
			Bio:      userResp.GetUserProfile().GetBio(),
			Gender:   int32(userResp.GetUserProfile().GetGender()),
		},
		UserProfileCounts: types.UserProfileCounts{
			FolloweeCount:         countResp.GetFollowingCount(),
			FollowerCount:         countResp.GetFollowedCount(),
			LikeReceivedCount:     countResp.GetLikeCount(),
			FavoriteReceivedCount: countResp.GetFavoriteCount(),
		},
		ViewerProfileState: types.ViewerProfileState{
			IsFollowing: false,
		},
	}, nil
}
