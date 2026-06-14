package recommend

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func RecallInterest(
	ctx context.Context,
	rds *redis.Redis,
	userID int64,
	cfg contentconfig.RecommendInterestConfig,
) ([]int64, error) {
	if rds == nil || userID <= 0 || !cfg.Enabled {
		return []int64{}, nil
	}
	cfg = NormalizeConfig(contentconfig.RecommendConfig{Interest: cfg}).Interest

	profile, err := LoadUserProfile(ctx, rds, userID, cfg)
	if err != nil || len(profile) == 0 {
		return []int64{}, err
	}

	scoreByID := make(map[int64]float64)
	for tag, profileWeight := range profile {
		pairs, err := rds.ZrevrangeWithScoresByFloatCtx(
			ctx,
			redisconsts.BuildRecommendTagIndexKey(tag),
			0,
			int64(cfg.Limit-1),
		)
		if err != nil {
			return nil, err
		}
		for rank, pair := range pairs {
			id, err := strconv.ParseInt(pair.Key, 10, 64)
			if err != nil || id <= 0 {
				continue
			}
			scoreByID[id] += profileWeight * rankScore(rank)
		}
	}

	return rankIDs(scoreByID, cfg.Limit), nil
}
