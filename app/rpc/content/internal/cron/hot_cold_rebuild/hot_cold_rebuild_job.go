package hot_cold_rebuild

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/hotrank"
	"zfeed/pkg/xxljob"
)

const HandlerName = "hot.cold.rebuild"

const (
	defaultWindowDays  = 15
	defaultTopN        = 5000
	defaultLockTTL     = 3600
	defaultBatchSize   = 500
	defaultPageSize    = 1000
	defaultHalfLife    = 24
	defaultSnapshotTTL = 3600
	snapshotIDLayout   = "20060102150405"
	lockDateLayout     = "20060102"
)

type Params struct {
	WindowDays    int              `json:"windowDays"`
	TopN          int              `json:"topN"`
	LockTTL       int              `json:"lockTtl"`
	HalfLifeHours float64          `json:"halfLifeHours"`
	SnapshotTTL   int              `json:"snapshotTtl"`
	Weights       *hotrank.Weights `json:"weights"`
	BatchSize     int              `json:"batchSize"`
	PageSize      int              `json:"pageSize"`
}

type HotColdRebuildJob struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
}

func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &HotColdRebuildJob{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

func (j *HotColdRebuildJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	p := parseParams(param.ExecutorParams)
	if p.WindowDays <= 0 {
		p.WindowDays = defaultWindowDays
	}
	if p.TopN <= 0 {
		p.TopN = defaultTopN
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}
	if p.BatchSize <= 0 {
		p.BatchSize = defaultBatchSize
	}
	if p.PageSize <= 0 {
		p.PageSize = defaultPageSize
	}
	if p.HalfLifeHours <= 0 {
		p.HalfLifeHours = defaultHalfLife
	}
	if p.SnapshotTTL <= 0 {
		p.SnapshotTTL = defaultSnapshotTTL
	}

	formula := hotrank.Formula{
		Weights:       mergeWeights(p.Weights),
		HalfLifeHours: p.HalfLifeHours,
	}

	now := time.Now().UTC()
	lockKey := redisconsts.BuildHotFeedColdLockKey(now.Format(lockDateLayout))
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

	if _, err := j.svc.Redis.DelCtx(ctx, redisconsts.RedisFeedHotGlobalKey); err != nil {
		return "", err
	}

	startTime := now.Add(-time.Duration(p.WindowDays) * 24 * time.Hour)
	if err := j.rebuildFromDB(ctx, formula, startTime, now, p); err != nil {
		return "", err
	}

	pairs, err := j.svc.Redis.ZrevrangeWithScoresByFloatCtx(ctx, redisconsts.RedisFeedHotGlobalKey, 0, int64(p.TopN-1))
	if err != nil {
		return "", err
	}
	if len(pairs) > 0 {
		snapshotID := now.Format(snapshotIDLayout)
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

func (j *HotColdRebuildJob) rebuildFromDB(ctx context.Context, formula hotrank.Formula, startTime, now time.Time, p Params) error {
	cursorID := int64(0)
	for {
		rows, err := j.contentRepo.ListColdUpdateContents(30, 10, startTime, cursorID, p.PageSize)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		ids := make([]int64, 0, len(rows))
		scores := make([]float64, 0, len(rows))
		redisArgs := make([]interface{}, 0, len(rows)*2)
		for _, row := range rows {
			if row == nil || row.PublishedAt == nil {
				continue
			}
			score := calcScore(formula, row, now)
			ids = append(ids, row.ID)
			scores = append(scores, score)
			redisArgs = append(redisArgs, score, strconv.FormatInt(row.ID, 10))
		}

		if len(ids) > 0 {
			if err := j.batchUpdateHotScore(ids, scores, p.BatchSize); err != nil {
				return err
			}
			if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotFeedZSetScript, []string{
				redisconsts.RedisFeedHotGlobalKey,
			}, redisArgs...); err != nil {
				return err
			}
		}

		cursorID = rows[len(rows)-1].ID
		if len(rows) < p.PageSize {
			return nil
		}
	}
}

func calcScore(formula hotrank.Formula, row *model.ZfeedContent, now time.Time) float64 {
	publishedAt := now
	if row.PublishedAt != nil {
		publishedAt = row.PublishedAt.UTC()
	}
	return formula.Score(row.LikeCount, row.CommentCount, row.FavoriteCount, publishedAt, now)
}

func (j *HotColdRebuildJob) batchUpdateHotScore(ids []int64, scores []float64, batchSize int) error {
	if len(ids) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := j.contentRepo.BatchUpdateHotScores(ids[start:end], scores[start:end], time.Now()); err != nil {
			return err
		}
	}
	return nil
}
