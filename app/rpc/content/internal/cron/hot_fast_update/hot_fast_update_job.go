package hot_fast_update

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/hotrank"
	"zfeed/pkg/xxljob"
)

const HandlerName = "hot.fast.update"

const (
	defaultShards       = redisconsts.RedisFeedHotIncDefaultShards
	defaultTopN         = 5000
	defaultLockTTL      = 300
	defaultHalfLifeHour = 24
	defaultSnapshotTTL  = 3600
	snapshotIDLayout    = "20060102150405"
	bucketLayout        = "200601021504"
)

type Params struct {
	Shards        int              `json:"shards"`
	TopN          int              `json:"topN"`
	LockTTL       int              `json:"lockTtl"`
	HalfLifeHours float64          `json:"halfLifeHours"`
	SnapshotTTL   int              `json:"snapshotTtl"`
	Weights       *hotrank.Weights `json:"weights"`
}

type ActionCounts struct {
	Like     int64 `json:"like"`
	Comment  int64 `json:"comment"`
	Favorite int64 `json:"favorite"`
	TS       int64 `json:"ts"`
}

type HotFastUpdateJob struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
}

func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &HotFastUpdateJob{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

func (j *HotFastUpdateJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	p := parseParams(param.ExecutorParams)
	if p.Shards <= 0 {
		p.Shards = defaultShards
	}
	if p.TopN <= 0 {
		p.TopN = defaultTopN
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}
	if p.HalfLifeHours <= 0 {
		p.HalfLifeHours = defaultHalfLifeHour
	}
	if p.SnapshotTTL <= 0 {
		p.SnapshotTTL = defaultSnapshotTTL
	}

	formula := hotrank.Formula{
		Weights:       mergeWeights(p.Weights),
		HalfLifeHours: p.HalfLifeHours,
	}

	bucket := time.Now().UTC().Format(bucketLayout)
	lockKey := redisconsts.BuildHotFeedFastLockKey(bucket)
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

	updatedScores := make(map[int64]float64)
	for shard := 0; shard < p.Shards; shard++ {
		incKey := redisconsts.BuildHotFeedIncKey(shard)
		now := time.Now().UTC()
		items, err := j.svc.Redis.HgetallCtx(ctx, incKey)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			continue
		}

		deltaMap := make(map[string]float64, len(items))
		for itemID, raw := range items {
			delta, err := computeDelta(raw, formula, now)
			if err != nil {
				return "", err
			}
			if delta == 0 {
				continue
			}
			deltaMap[itemID] = delta
		}
		if len(deltaMap) == 0 {
			continue
		}
		if err := j.mergeIncAtomic(ctx, incKey, deltaMap, updatedScores); err != nil {
			return "", err
		}
	}

	pairs, err := j.svc.Redis.ZrevrangeWithScoresByFloatCtx(ctx, redisconsts.RedisFeedHotGlobalKey, 0, int64(p.TopN-1))
	if err != nil {
		return "", err
	}
	if err := j.flushHotScoresTopN(ctx, pairs, updatedScores); err != nil {
		return "", err
	}

	if len(pairs) > 0 {
		snapshotID := time.Now().UTC().Format(snapshotIDLayout)
		snapshotKey := redisconsts.BuildHotFeedSnapshotKey(snapshotID)
		if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotSnapshotScript, []string{
			redisconsts.RedisFeedHotGlobalKey,
			snapshotKey,
			redisconsts.RedisFeedHotGlobalLatestKey,
		}, strconv.Itoa(p.TopN), snapshotID, strconv.Itoa(p.SnapshotTTL)); err != nil {
			return "", err
		}
	}

	return "ok", nil
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

func mergeWeights(w *hotrank.Weights) hotrank.Weights {
	base := hotrank.DefaultWeights()
	if w == nil {
		return base
	}
	if w.Like > 0 {
		base.Like = w.Like
	}
	if w.Comment > 0 {
		base.Comment = w.Comment
	}
	if w.Favorite > 0 {
		base.Favorite = w.Favorite
	}
	return base
}

func computeDelta(raw string, formula hotrank.Formula, now time.Time) (float64, error) {
	if v, err := strconv.ParseFloat(raw, 64); err == nil {
		if v <= 0 {
			return 0, nil
		}
		return hotrank.Round3(v), nil
	}

	var payload ActionCounts
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0, err
	}
	if payload.Like == 0 && payload.Comment == 0 && payload.Favorite == 0 {
		return 0, nil
	}

	eventTime := now
	if payload.TS > 0 {
		eventTime = time.Unix(payload.TS, 0).UTC()
	}
	ageHours := now.Sub(eventTime).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	weighted := float64(payload.Like)*formula.Weights.Like +
		float64(payload.Comment)*formula.Weights.Comment +
		float64(payload.Favorite)*formula.Weights.Favorite
	if weighted <= 0 {
		return 0, nil
	}
	decay := 1.0
	if formula.HalfLifeHours > 0 {
		decay = math.Exp(-math.Ln2 * ageHours / formula.HalfLifeHours)
	}
	return hotrank.Round3(math.Log1p(weighted) * decay), nil
}

func (j *HotFastUpdateJob) flushHotScoresTopN(ctx context.Context, pairs []redis.FloatPair, updated map[int64]float64) error {
	if len(pairs) == 0 || len(updated) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(pairs))
	values := make([]float64, 0, len(pairs))
	for _, pair := range pairs {
		id, err := strconv.ParseInt(pair.Key, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid content id: %s", pair.Key)
		}
		if _, ok := updated[id]; !ok {
			continue
		}
		ids = append(ids, id)
		values = append(values, pair.Score)
	}
	if len(ids) == 0 {
		return nil
	}
	return j.contentRepo.BatchUpdateHotScores(ids, values, time.Now())
}

func (j *HotFastUpdateJob) mergeIncAtomic(ctx context.Context, incKey string, deltaMap map[string]float64, updated map[int64]float64) error {
	args := make([]interface{}, 0, 1+len(deltaMap)*2)
	args = append(args, "3")
	for member, delta := range deltaMap {
		args = append(args, member, strconv.FormatFloat(delta, 'f', 6, 64))
	}

	if _, err := j.svc.Redis.EvalCtx(ctx, luautils.MergeHotIncScript, []string{
		incKey,
		redisconsts.RedisFeedHotGlobalKey,
	}, args...); err != nil {
		return err
	}

	for itemID := range deltaMap {
		score, err := j.svc.Redis.ZscoreByFloatCtx(ctx, redisconsts.RedisFeedHotGlobalKey, itemID)
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(itemID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid content id: %s", itemID)
		}
		updated[id] = score
	}
	return nil
}
