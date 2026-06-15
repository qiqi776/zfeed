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
	countKey := redisconsts.BuildRecommendSeenCountKey(userID)
	if _, err := rds.ZaddsCtx(ctx, key, pairs...); err != nil {
		return err
	}
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		if _, err := rds.HincrbyCtx(ctx, countKey, strconv.FormatInt(contentID, 10), 1); err != nil {
			return err
		}
	}
	if err := rds.ExpireCtx(ctx, key, cfg.SeenTTL); err != nil {
		return err
	}
	return rds.ExpireCtx(ctx, countKey, cfg.SeenTTL)
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

	fields := make([]string, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		fields = append(fields, strconv.FormatInt(contentID, 10))
	}
	if len(fields) == 0 {
		return result, nil
	}

	countKey := redisconsts.BuildRecommendSeenCountKey(userID)
	counts, err := rds.HmgetCtx(ctx, countKey, fields...)
	if err != nil && !isRedisNil(err) {
		return nil, err
	}

	key := redisconsts.BuildRecommendSeenKey(userID)
	for i, field := range fields {
		contentID, _ := strconv.ParseInt(field, 10, 64)
		if i < len(counts) && counts[i] != "" {
			count, parseErr := strconv.Atoi(counts[i])
			if parseErr != nil {
				return nil, parseErr
			}
			if count > 0 {
				result[contentID] = count
			}
			continue
		}

		score, err := rds.ZscoreCtx(ctx, key, field)
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
