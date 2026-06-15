package rec_tag_refresh

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/hotrank"
	"zfeed/pkg/xxljob"
)

const HandlerName = "rec.tag.refresh"

const (
	defaultWindowDays = 7
	defaultPageSize   = 500
	defaultBatchSize  = 200
	defaultLockTTL    = 1800
	defaultHalfLife   = 24
	lockBucketLayout  = "200601021504"
	publishedStatus   = 30
	publicVisibility  = 10
)

type Params struct {
	WindowDays    int     `json:"windowDays"`
	PageSize      int     `json:"pageSize"`
	BatchSize     int     `json:"batchSize"`
	LockTTL       int     `json:"lockTtl"`
	HalfLifeHours float64 `json:"halfLifeHours"`
}

type RecTagRefreshJob struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
	now         func() time.Time
}

func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &RecTagRefreshJob{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

func (j *RecTagRefreshJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	if j == nil || j.svc == nil || j.svc.Redis == nil || j.contentRepo == nil {
		return "", fmt.Errorf("rec tag refresh dependencies are not ready")
	}

	p := normalizeParams(parseParams(param.ExecutorParams))
	now := time.Now().UTC()
	if j.now != nil {
		now = j.now().UTC()
	}

	lockKey := redisconsts.BuildRecommendTagRefreshLockKey(now.Format(lockBucketLayout))
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

	start := now.Add(-time.Duration(p.WindowDays) * 24 * time.Hour)
	formula := hotrank.Formula{
		Weights:       hotrank.DefaultWeights(),
		HalfLifeHours: p.HalfLifeHours,
	}
	refreshed, err := j.refresh(ctx, formula, start, now, p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ok refreshed=%d", refreshed), nil
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

func normalizeParams(p Params) Params {
	if p.WindowDays <= 0 {
		p.WindowDays = defaultWindowDays
	}
	if p.PageSize <= 0 {
		p.PageSize = defaultPageSize
	}
	if p.BatchSize <= 0 {
		p.BatchSize = defaultBatchSize
	}
	if p.LockTTL <= 0 {
		p.LockTTL = defaultLockTTL
	}
	if p.HalfLifeHours <= 0 {
		p.HalfLifeHours = defaultHalfLife
	}
	return p
}

func (j *RecTagRefreshJob) refresh(
	ctx context.Context,
	formula hotrank.Formula,
	start time.Time,
	now time.Time,
	p Params,
) (int, error) {
	var refreshed int
	cursorID := int64(0)
	for {
		rows, err := j.contentRepo.ListColdUpdateContents(
			publishedStatus,
			publicVisibility,
			start,
			cursorID,
			p.PageSize,
		)
		if err != nil {
			return refreshed, err
		}
		if len(rows) == 0 {
			return refreshed, nil
		}

		for _, row := range rows {
			ok, err := j.refreshRow(ctx, formula, row, now)
			if err != nil {
				return refreshed, err
			}
			if ok {
				refreshed++
			}
		}

		cursorID = rows[len(rows)-1].ID
		if len(rows) < p.PageSize {
			return refreshed, nil
		}
	}
}

func (j *RecTagRefreshJob) refreshRow(
	ctx context.Context,
	formula hotrank.Formula,
	row *model.ZfeedContent,
	now time.Time,
) (bool, error) {
	if row == nil || row.PublishedAt == nil {
		return false, nil
	}
	score := tagIndexBaseScore(
		formula,
		row,
		now,
	)
	return recommend.RefreshContentTagIndex(
		ctx,
		j.svc.Redis,
		j.svc.Config.Recommend,
		row.ID,
		score,
	)
}

func tagIndexBaseScore(formula hotrank.Formula, row *model.ZfeedContent, now time.Time) float64 {
	if row == nil || row.PublishedAt == nil {
		return 0
	}

	hotScore := formula.Score(
		row.LikeCount,
		row.CommentCount,
		row.FavoriteCount,
		row.PublishedAt.UTC(),
		now,
	)
	freshnessScore := tagIndexFreshnessScore(row.PublishedAt.UTC(), now, formula.HalfLifeHours)
	if hotScore <= 0 {
		return freshnessScore
	}
	return hotrank.Round3(hotScore + freshnessScore)
}

func tagIndexFreshnessScore(publishedAt, now time.Time, halfLifeHours float64) float64 {
	if halfLifeHours <= 0 {
		halfLifeHours = defaultHalfLife
	}
	ageHours := now.Sub(publishedAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	return hotrank.Round3(math.Exp(-math.Ln2 * ageHours / halfLifeHours))
}
