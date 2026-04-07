package logic

import (
	"context"
	"encoding/json"

	"zfeed/app/rpc/count/count"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserProfileCountsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

func NewGetUserProfileCountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserProfileCountsLogic {
	return &GetUserProfileCountsLogic{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetUserProfileCountsLogic) GetUserProfileCounts(in *count.GetUserProfileCountsReq) (*count.GetUserProfileCountsRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	cacheKey := buildUserProfileCountsCacheKey(in.GetUserId())
	cacheStr, err := l.svcCtx.Redis.GetCtx(l.ctx, cacheKey)
	if err == nil && cacheStr != "" {
		var cached userProfileCountsCache
		if unmarshalErr := json.Unmarshal([]byte(cacheStr), &cached); unmarshalErr == nil {
			return &count.GetUserProfileCountsRes{
				FollowingCount: cached.FollowingCount,
				FollowedCount:  cached.FollowedCount,
				LikeCount:      cached.LikeCount,
				FavoriteCount:  cached.FavoriteCount,
			}, nil
		}
	}

	likeCount, err := l.countRepo.SumByOwner(int32(count.BizType_LIKE), int32(count.TargetType_CONTENT), in.GetUserId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询主页计数失败"))
	}
	favoriteCount, err := l.countRepo.SumByOwner(int32(count.BizType_FAVORITE), int32(count.TargetType_CONTENT), in.GetUserId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询主页计数失败"))
	}

	var followingCount int64
	if row, getErr := l.countRepo.Get(int32(count.BizType_FOLLOWING), int32(count.TargetType_USER), in.GetUserId()); getErr != nil {
		return nil, errorx.Wrap(l.ctx, getErr, errorx.NewMsg("查询主页计数失败"))
	} else if row != nil {
		followingCount = row.Value
	}

	var followedCount int64
	if row, getErr := l.countRepo.Get(int32(count.BizType_FOLLOWED), int32(count.TargetType_USER), in.GetUserId()); getErr != nil {
		return nil, errorx.Wrap(l.ctx, getErr, errorx.NewMsg("查询主页计数失败"))
	} else if row != nil {
		followedCount = row.Value
	}

	resp := &count.GetUserProfileCountsRes{
		FollowingCount: followingCount,
		FollowedCount:  followedCount,
		LikeCount:      likeCount,
		FavoriteCount:  favoriteCount,
	}
	payload, marshalErr := marshalUserProfileCounts(resp)
	if marshalErr == nil {
		if err := l.svcCtx.Redis.SetexCtx(
			l.ctx,
			cacheKey,
			payload,
			countCacheExpireSecondsWithJitter(redisconsts.RedisUserProfileCountsExpireSeconds),
		); err != nil {
			l.Errorf("set user profile counts cache failed, key=%s, err=%v", cacheKey, err)
		}
	}

	return resp, nil
}
