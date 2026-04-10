package hot_snapshot_refresh

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/xxljob"
)

const HandlerName = "hot.snapshot.refresh"

const (
	defaultTopN        = 5000
	defaultSnapshotTTL = 3600
	snapshotIDLayout   = "20060102150405"
)

type Params struct {
	TopN        int `json:"topN"`
	SnapshotTTL int `json:"snapshotTtl"`
}

type HotSnapshotRefreshJob struct {
	svc *svc.ServiceContext
}

func Register(_ context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &HotSnapshotRefreshJob{svc: svcCtx}
	executor.RegisterTask(HandlerName, job.Run)
}

func (j *HotSnapshotRefreshJob) Run(ctx context.Context, param xxljob.TriggerParam) (string, error) {
	p := parseParams(param.ExecutorParams)
	if p.TopN <= 0 {
		p.TopN = defaultTopN
	}
	if p.SnapshotTTL <= 0 {
		p.SnapshotTTL = defaultSnapshotTTL
	}

	snapshotID := time.Now().UTC().Format(snapshotIDLayout)
	snapshotKey := redisconsts.BuildHotFeedSnapshotKey(snapshotID)
	res, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotSnapshotScript, []string{
		redisconsts.RedisFeedHotGlobalKey,
		snapshotKey,
		redisconsts.RedisFeedHotGlobalLatestKey,
	}, strconv.Itoa(p.TopN), snapshotID, strconv.Itoa(p.SnapshotTTL))
	if err != nil {
		return "", err
	}
	count := luaResultCount(res)
	return fmt.Sprintf("ok snapshot=%s count=%d", snapshotID, count), nil
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

func luaResultCount(res any) int64 {
	arr, ok := res.([]interface{})
	if !ok || len(arr) == 0 {
		return 0
	}
	switch v := arr[0].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case []byte:
		n, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return 0
		}
		return n
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
