package rec_new_cleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/redisx"
	"zfeed/pkg/xxljob"
)

const HandlerName = "rec.new.cleanup"

const (
	defaultMaxAgeHours = 7 * 24
	defaultBatchSize   = 200
	defaultLockTTL     = 3600
	lockBucketLayout   = "2006010215"
)

type Params struct {
	MaxAgeHours int `json:"maxAgeHours"`
	BatchSize   int `json:"batchSize"`
	LockTTL     int `json:"lockTtl"`
}

type RecNewCleanupJob struct {
	svc *svc.ServiceContext
	now func() time.Time
}

func Register(_ context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &RecNewCleanupJob{svc: svcCtx}
	executor.RegisterTask(HandlerName, job.Run)
}

func (j *RecNewCleanupJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	if j == nil || j.svc == nil || j.svc.Redis == nil {
		return "", fmt.Errorf("rec new cleanup dependencies are not ready")
	}

	p := parseParams(param.ExecutorParams)
	if p.MaxAgeHours <= 0 {
		p.MaxAgeHours = defaultMaxAgeHours
	}
	if p.BatchSize <= 0 {
		p.BatchSize = defaultBatchSize
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}

	now := time.Now().UTC()
	if j.now != nil {
		now = j.now().UTC()
	}
	lockKey := redisconsts.BuildRecommendNewCleanupLockKey(now.Format(lockBucketLayout))
	redisLock := redis.NewRedisLock(j.svc.Redis, lockKey)
	redisLock.SetExpire(p.LockTTL)
	locked, err := redisLock.AcquireCtx(ctx)
	if err != nil {
		return "", err
	}
	if !locked {
		return "duplicate", nil
	}
	defer redisLock.ReleaseCtx(context.Background())

	cutoff := now.Add(-time.Duration(p.MaxAgeHours) * time.Hour).Unix()
	removed, err := j.cleanupExpired(ctx, cutoff, p.BatchSize)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ok removed=%d", removed), nil
}

func (j *RecNewCleanupJob) cleanupExpired(ctx context.Context, cutoff int64, batchSize int) (int, error) {
	var removed int
	for {
		pairs, err := redisx.ZRangeByScoreWithScoresAndLimitCtx(
			ctx,
			j.svc.Redis,
			redisconsts.RecommendNewContentKey,
			"0",
			strconv.FormatInt(cutoff, 10),
			0,
			batchSize,
		)
		if err != nil {
			return removed, err
		}
		if len(pairs) == 0 {
			return removed, nil
		}

		for _, pair := range pairs {
			contentID, err := strconv.ParseInt(pair.Key, 10, 64)
			if err != nil || contentID <= 0 {
				if _, zremErr := j.svc.Redis.ZremCtx(ctx, redisconsts.RecommendNewContentKey, pair.Key); zremErr != nil {
					return removed, zremErr
				}
				removed++
				continue
			}
			if err := j.cleanupContent(ctx, contentID, pair.Key); err != nil {
				return removed, err
			}
			removed++
		}
	}
}

func (j *RecNewCleanupJob) cleanupContent(ctx context.Context, contentID int64, member string) error {
	tagsKey := redisconsts.BuildRecommendContentTagsKey(contentID)
	tags, err := j.svc.Redis.HgetallCtx(ctx, tagsKey)
	if err != nil {
		return err
	}
	for tag := range tags {
		if tag == "" {
			continue
		}
		if _, err := j.svc.Redis.ZremCtx(ctx, redisconsts.BuildRecommendTagIndexKey(tag), member); err != nil {
			return err
		}
	}
	if _, err := j.svc.Redis.DelCtx(ctx,
		redisconsts.BuildRecommendNewContentMetaKey(contentID),
		tagsKey,
	); err != nil {
		return err
	}
	if _, err := j.svc.Redis.ZremCtx(ctx, redisconsts.RecommendNewContentKey, member); err != nil {
		return err
	}
	return nil
}

func parseParams(raw string) Params {
	if raw == "" {
		return Params{}
	}
	var p Params
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return Params{}
	}
	return p
}
