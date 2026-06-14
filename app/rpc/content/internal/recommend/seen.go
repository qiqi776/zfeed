package recommend

import (
	"context"
	"strconv"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func RecordSeenContents(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	userID int64,
	contentIDs []int64,
	now time.Time,
) error {
	if rds == nil || userID <= 0 || len(contentIDs) == 0 {
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	cfg = NormalizeConfig(cfg)

	pairs := make([]gzredis.Pair, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		pairs = append(pairs, gzredis.Pair{
			Key:   strconv.FormatInt(contentID, 10),
			Score: now.Unix(),
		})
	}
	if len(pairs) == 0 {
		return nil
	}

	key := redisconsts.BuildRecommendSeenKey(userID)
	if _, err := rds.ZaddsCtx(ctx, key, pairs...); err != nil {
		return err
	}
	return rds.ExpireCtx(ctx, key, cfg.SeenTTL)
}

func LoadSeenCounts(
	ctx context.Context,
	rds *gzredis.Redis,
	userID int64,
	contentIDs []int64,
) (map[int64]int, error) {
	result := make(map[int64]int)
	if rds == nil || userID <= 0 || len(contentIDs) == 0 {
		return result, nil
	}

	key := redisconsts.BuildRecommendSeenKey(userID)
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		score, err := rds.ZscoreCtx(ctx, key, strconv.FormatInt(contentID, 10))
		if err != nil {
			if isRedisNil(err) {
				continue
			}
			return nil, err
		}
		if score != 0 {
			result[contentID] = 1
		}
	}
	return result, nil
}
