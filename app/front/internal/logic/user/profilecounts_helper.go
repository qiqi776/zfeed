package user

import (
	"context"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/rpc/count/count"
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

func defaultUserProfileCounts() *count.GetUserProfileCountsRes {
	return &count.GetUserProfileCountsRes{}
}
