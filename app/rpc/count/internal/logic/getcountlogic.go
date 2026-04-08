package logic

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"zfeed/app/rpc/count/count"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	redislock "github.com/zeromicro/go-zero/core/stores/redis"
)

type GetCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

func NewGetCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCountLogic {
	return &GetCountLogic{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetCountLogic) GetCount(in *count.GetCountReq) (*count.GetCountRes, error) {
	if in == nil || in.GetTargetId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetBizType() == count.BizType_BIZ_TYPE_UNKNOWN || in.GetTargetType() == count.TargetType_TARGET_TYPE_UNKNOWN {
		return nil, errorx.NewMsg("参数错误")
	}

	cacheKey := buildCountValueCacheKey(in.GetBizType(), in.GetTargetType(), in.GetTargetId())
	cacheValue, cacheResult := l.queryFromCache(cacheKey)
	if cacheResult == cacheHit {
		return &count.GetCountRes{Value: cacheValue}, nil
	}

	value, err := l.rebuildCacheWithLock(in, cacheKey)
	if err != nil {
		return nil, err
	}
	return &count.GetCountRes{Value: value}, nil
}

func (l *GetCountLogic) queryFromCache(cacheKey string) (int64, cacheQueryResult) {
	cacheStr, err := l.svcCtx.Redis.GetCtx(l.ctx, cacheKey)
	if err != nil {
		l.Errorf("query count cache failed, key=%s, err=%v", cacheKey, err)
		return 0, cacheError
	}
	if cacheStr == "" {
		return 0, cacheMiss
	}

	value, parseErr := strconv.ParseInt(cacheStr, 10, 64)
	if parseErr != nil {
		l.Errorf("parse count cache failed, key=%s, value=%s, err=%v", cacheKey, cacheStr, parseErr)
		return 0, cacheError
	}
	return value, cacheHit
}

func (l *GetCountLogic) rebuildCacheWithLock(in *count.GetCountReq, cacheKey string) (int64, error) {
	lockKey := buildCountValueRebuildLockKey(in.GetBizType(), in.GetTargetType(), in.GetTargetId())
	lock := redislock.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(rebuildLockExpireSeconds)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		l.Errorf("acquire count rebuild lock failed, lock_key=%s, err=%v", lockKey, err)
		return l.queryFromDB(in)
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
				return 0, l.ctx.Err()
			default:
			}
			time.Sleep(time.Duration(baseSleepMs+rand.Intn(jitterMs)) * time.Millisecond)

			if value, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
				return value, nil
			}
		}
		return l.queryFromDB(in)
	}

	defer func() {
		if releaseOK, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOK || releaseErr != nil {
			l.Errorf("release count rebuild lock failed, lock_key=%s, err=%v", lockKey, releaseErr)
		}
	}()

	if value, cacheResult := l.queryFromCache(cacheKey); cacheResult == cacheHit {
		return value, nil
	}

	value, err := l.queryFromDB(in)
	if err != nil {
		return 0, err
	}

	if err := l.svcCtx.Redis.SetexCtx(
		l.ctx,
		cacheKey,
		strconv.FormatInt(value, 10),
		countCacheExpireSecondsWithJitter(redisconsts.RedisCountValueExpireSeconds),
	); err != nil {
		l.Errorf("rebuild count cache failed, key=%s, err=%v", cacheKey, err)
	}

	return value, nil
}

func (l *GetCountLogic) queryFromDB(in *count.GetCountReq) (int64, error) {
	row, err := l.countRepo.Get(int32(in.GetBizType()), int32(in.GetTargetType()), in.GetTargetId())
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询计数失败"))
	}

	value := int64(0)
	if row != nil {
		value = row.Value
	}
	return value, nil
}
