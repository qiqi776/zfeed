package logic

import (
	"context"
	"math/rand"
	"time"

	"zfeed/app/rpc/count/count"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	redislock "github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	userProfileContentStatusPublished  = 30
	userProfileContentVisibilityPublic = 10
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
		return nil, errorx.NewBadRequest("参数错误")
	}

	cacheKey := buildUserProfileCountsCacheKey(in.GetUserId())
	if cacheValue, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
		return l.attachContentCount(cacheValue, in.GetUserId())
	}

	resp, err := l.rebuildCacheWithLock(in.GetUserId(), cacheKey)
	if err != nil {
		return nil, err
	}
	return l.attachContentCount(resp, in.GetUserId())
}

func (l *GetUserProfileCountsLogic) queryFromCache(cacheKey string) (*count.GetUserProfileCountsRes, cacheQueryResult) {
	cacheStr, err := l.svcCtx.Redis.GetCtx(l.ctx, cacheKey)
	if err != nil {
		l.Errorf("query user profile counts cache failed, key=%s, err=%v", cacheKey, err)
		return nil, cacheError
	}
	if cacheStr == "" {
		return nil, cacheMiss
	}

	value, err := unmarshalUserProfileCounts(cacheStr)
	if err != nil {
		l.Errorf("parse user profile counts cache failed, key=%s, value=%s, err=%v", cacheKey, cacheStr, err)
		return nil, cacheError
	}
	return value, cacheHit
}

func (l *GetUserProfileCountsLogic) rebuildCacheWithLock(userID int64, cacheKey string) (*count.GetUserProfileCountsRes, error) {
	lockKey := buildUserProfileCountsRebuildLockKey(userID)
	lock := redislock.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(rebuildLockExpireSeconds)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		l.Errorf("acquire user profile counts rebuild lock failed, lock_key=%s, err=%v", lockKey, err)
		return l.queryFromDB(userID)
	}

	if !lockAcquired {
		const (
			maxRetry    = 5
			baseSleepMs = 30
			jitterMs    = 50
		)
		for i := 0; i < maxRetry; i++ {
			select {
			case <-l.ctx.Done():
				return nil, l.ctx.Err()
			default:
			}
			time.Sleep(time.Duration(baseSleepMs+rand.Intn(jitterMs)) * time.Millisecond)

			if value, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
				return value, nil
			}
		}
		return l.queryFromDB(userID)
	}

	defer func() {
		if releaseOK, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOK || releaseErr != nil {
			l.Errorf("release user profile counts rebuild lock failed, lock_key=%s, err=%v", lockKey, releaseErr)
		}
	}()

	if value, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
		return value, nil
	}

	resp, err := l.queryFromDB(userID)
	if err != nil {
		return nil, err
	}

	payload, marshalErr := marshalUserProfileCounts(resp)
	if marshalErr == nil {
		if err := l.svcCtx.Redis.SetexCtx(
			l.ctx,
			cacheKey,
			payload,
			countCacheExpireSecondsWithJitter(redisconsts.RedisUserProfileCountsExpireSeconds),
		); err != nil {
			l.Errorf("rebuild user profile counts cache failed, key=%s, err=%v", cacheKey, err)
		}
	} else {
		l.Errorf("marshal user profile counts cache failed, user_id=%d, err=%v", userID, marshalErr)
	}

	return resp, nil
}

func (l *GetUserProfileCountsLogic) queryFromDB(userID int64) (*count.GetUserProfileCountsRes, error) {
	if userID <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	likeCount, err := l.countRepo.SumByOwner(int32(count.BizType_LIKE), int32(count.TargetType_CONTENT), userID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询主页计数失败"))
	}
	favoriteCount, err := l.countRepo.SumByOwner(int32(count.BizType_FAVORITE), int32(count.TargetType_CONTENT), userID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询主页计数失败"))
	}

	var followingCount int64
	if row, getErr := l.countRepo.Get(int32(count.BizType_FOLLOWING), int32(count.TargetType_USER), userID); getErr != nil {
		return nil, errorx.Wrap(l.ctx, getErr, errorx.NewMsg("查询主页计数失败"))
	} else if row != nil {
		followingCount = row.Value
	}

	var followedCount int64
	if row, getErr := l.countRepo.Get(int32(count.BizType_FOLLOWED), int32(count.TargetType_USER), userID); getErr != nil {
		return nil, errorx.Wrap(l.ctx, getErr, errorx.NewMsg("查询主页计数失败"))
	} else if row != nil {
		followedCount = row.Value
	}

	return &count.GetUserProfileCountsRes{
		FollowingCount: followingCount,
		FollowedCount:  followedCount,
		LikeCount:      likeCount,
		FavoriteCount:  favoriteCount,
	}, nil
}

func (l *GetUserProfileCountsLogic) attachContentCount(resp *count.GetUserProfileCountsRes, userID int64) (*count.GetUserProfileCountsRes, error) {
	if resp == nil {
		resp = &count.GetUserProfileCountsRes{}
	}

	contentCount, err := l.queryContentCount(userID)
	if err != nil {
		return nil, err
	}
	resp.ContentCount = contentCount
	return resp, nil
}

func (l *GetUserProfileCountsLogic) queryContentCount(userID int64) (int64, error) {
	if userID <= 0 || l.svcCtx == nil || l.svcCtx.MysqlDb == nil {
		return 0, nil
	}

	var value int64
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_content").
		Where(
			"user_id = ? AND status = ? AND visibility = ? AND is_deleted = 0",
			userID,
			userProfileContentStatusPublished,
			userProfileContentVisibilityPublic,
		).
		Count(&value).Error
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询主页计数失败"))
	}
	return value, nil
}
