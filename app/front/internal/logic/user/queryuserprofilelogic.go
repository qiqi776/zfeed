// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"errors"
	"sync"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/count/count"
	followservicepb "zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
		userResp        *user.GetUserProfileRes
		userErr         error
		countResp       *count.GetUserProfileCountsRes
		countErr        error
		followResp      *followservicepb.GetFollowSummaryRes
		followErr       error
		contentCount    int64
		contentCountErr error
		wg              sync.WaitGroup
	)

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)

	wg.Add(4)

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
		if l.svcCtx == nil || l.svcCtx.FollowRpc == nil {
			followResp = &followservicepb.GetFollowSummaryRes{}
			return
		}
		followResp, followErr = l.svcCtx.FollowRpc.GetFollowSummary(l.ctx, &followservicepb.GetFollowSummaryReq{
			UserId:   req.UserId,
			ViewerId: &viewerID,
		})
	}()

	go func() {
		defer wg.Done()
		contentCount, contentCountErr = l.queryContentCount(req.UserId)
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
		followResp = &followservicepb.GetFollowSummaryRes{}
	}
	if followResp == nil {
		followResp = &followservicepb.GetFollowSummaryRes{}
	}
	if contentCountErr != nil {
		l.Errorf("query user content count failed, user_id=%d, err=%v", req.UserId, contentCountErr)
		contentCount = 0
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
			ContentCount:          contentCount,
		},
		ViewerProfileState: types.ViewerProfileState{
			IsFollowing: followResp.GetIsFollowing(),
		},
	}, nil
}

func (l *QueryUserProfileLogic) queryContentCount(userID int64) (int64, error) {
	if l.svcCtx == nil || l.svcCtx.MysqlDb == nil {
		return 0, nil
	}

	var countValue int64
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_content").
		Where("user_id = ? AND status = ? AND visibility = ? AND is_deleted = 0", userID, 30, 10).
		Count(&countValue).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return countValue, nil
}
