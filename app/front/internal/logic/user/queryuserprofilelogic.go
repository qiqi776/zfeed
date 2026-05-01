// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"sync"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/count/count"
	followservicepb "zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

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
		return nil, errorx.NewBadRequest("参数错误")
	}

	var (
		userResp   *user.GetUserProfileRes
		userErr    error
		countResp  *count.GetUserProfileCountsRes
		countErr   error
		followResp *followservicepb.GetFollowSummaryRes
		followErr  error
		wg         sync.WaitGroup
	)

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)

	wg.Add(3)

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

	go func() {
		defer wg.Done()
		followResp, followErr = loadFollowSummary(l.ctx, l.svcCtx, req.UserId, &viewerID)
	}()

	wg.Wait()

	if userErr != nil {
		return nil, userErr
	}
	if userResp.GetUserProfile() == nil {
		return nil, errorx.NewNotFound("用户不存在")
	}

	if countErr != nil {
		l.Errorf("query user profile counts failed, user_id=%d, err=%v", req.UserId, countErr)
		countResp = defaultUserProfileCounts()
	}
	if countResp == nil {
		countResp = defaultUserProfileCounts()
	}
	if followErr != nil {
		l.Errorf("query user follow summary failed, user_id=%d, viewer_id=%d, err=%v", req.UserId, viewerID, followErr)
		followResp = nil
	}
	followeeCount := resolveFolloweeCount(countResp, followResp)
	followerCount := resolveFollowerCount(countResp, followResp)
	isFollowing := false
	if followResp != nil {
		isFollowing = followResp.GetIsFollowing()
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
			FolloweeCount:         followeeCount,
			FollowerCount:         followerCount,
			LikeReceivedCount:     countResp.GetLikeCount(),
			FavoriteReceivedCount: countResp.GetFavoriteCount(),
			ContentCount:          countResp.GetContentCount(),
		},
		ViewerProfileState: types.ViewerProfileState{
			IsFollowing: isFollowing,
		},
	}, nil
}
