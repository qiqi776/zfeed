package user

import (
	"context"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/rpc/count/count"
	followservicepb "zfeed/app/rpc/interaction/client/followservice"
)

var defaultCountRPCTimeout = 200 * time.Millisecond

func getCountRPCTimeout(svcCtx *svc.ServiceContext) time.Duration {
	if svcCtx == nil || svcCtx.Config.CountRPCTimeoutMs <= 0 {
		return defaultCountRPCTimeout
	}
	return time.Duration(svcCtx.Config.CountRPCTimeoutMs) * time.Millisecond
}

func loadUserProfileCounts(ctx context.Context, svcCtx *svc.ServiceContext, userID int64) (*count.GetUserProfileCountsRes, error) {
	countCtx, cancel := context.WithTimeout(ctx, getCountRPCTimeout(svcCtx))
	defer cancel()

	return svcCtx.CountRpc.GetUserProfileCounts(countCtx, &count.GetUserProfileCountsReq{
		UserId: userID,
	})
}

func loadFollowSummary(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	userID int64,
	viewerID *int64,
) (*followservicepb.GetFollowSummaryRes, error) {
	if svcCtx == nil || svcCtx.FollowRpc == nil {
		return nil, nil
	}

	req := &followservicepb.GetFollowSummaryReq{
		UserId: userID,
	}
	if viewerID != nil && *viewerID > 0 {
		req.ViewerId = viewerID
	}

	return svcCtx.FollowRpc.GetFollowSummary(ctx, req)
}

func resolveFolloweeCount(
	countResp *count.GetUserProfileCountsRes,
	followResp *followservicepb.GetFollowSummaryRes,
) int64 {
	if followResp != nil {
		return followResp.GetFolloweeCount()
	}
	if countResp != nil {
		return countResp.GetFollowingCount()
	}
	return 0
}

func resolveFollowerCount(
	countResp *count.GetUserProfileCountsRes,
	followResp *followservicepb.GetFollowSummaryRes,
) int64 {
	if followResp != nil {
		return followResp.GetFollowerCount()
	}
	if countResp != nil {
		return countResp.GetFollowedCount()
	}
	return 0
}

func defaultUserProfileCounts() *count.GetUserProfileCountsRes {
	return &count.GetUserProfileCountsRes{}
}
