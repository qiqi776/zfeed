package hot_bucket_cleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/xxljob"
)

const HandlerName = "hot.bucket.cleanup"

const (
	defaultShards     = redisconsts.RedisFeedHotIncDefaultShards
	defaultLockTTL    = 3600
	cleanupDateLayout = "20060102"
)

type Params struct {
	Shards  int `json:"shards"`
	LockTTL int `json:"lockTtl"`
}

type HotBucketCleanupJob struct {
	svc *svc.ServiceContext
}

func Register(_ context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &HotBucketCleanupJob{svc: svcCtx}
	executor.RegisterTask(HandlerName, job.Run)
}

func (j *HotBucketCleanupJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	p := parseParams(param.ExecutorParams)
	if p.Shards <= 0 {
		p.Shards = defaultShards
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}

	lockDate := time.Now().UTC().Format(cleanupDateLayout)
	lockKey := redisconsts.BuildHotFeedBucketCleanupLockKey(lockDate)
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

	var removed int
	for shard := 0; shard < p.Shards; shard++ {
		incKey := redisconsts.BuildHotFeedIncKey(shard)
		count, err := j.svc.Redis.DelCtx(ctx, incKey)
		if err != nil {
			return "", err
		}
		removed += count
	}
	return fmt.Sprintf("ok removed=%d", removed), nil
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
