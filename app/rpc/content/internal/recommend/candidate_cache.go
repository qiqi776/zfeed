package recommend

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func BuildCandidateCacheKey(userID int64, variantID string, configHash string) string {
	return fmt.Sprintf(
		"%s:%04d:%s:%s",
		redisconsts.RecommendCandidatePrefix,
		userBucket(userID),
		normalizeCacheSegment(variantID, "control"),
		normalizeCacheSegment(configHash, "default"),
	)
}

func SaveCandidateCache(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	key string,
	candidates []Candidate,
) error {
	if rds == nil || strings.TrimSpace(key) == "" || len(candidates) == 0 {
		return nil
	}
	cfg = NormalizeConfig(cfg)

	pairs := make([]gzredis.Pair, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ContentID <= 0 {
			continue
		}
		pairs = append(pairs, gzredis.Pair{
			Key:   strconv.FormatInt(candidate.ContentID, 10),
			Score: int64(candidate.Score * candidateCacheScoreScale),
		})
	}
	if len(pairs) == 0 {
		return nil
	}
	if _, err := rds.ZaddsCtx(ctx, key, pairs...); err != nil {
		return err
	}
	return rds.ExpireCtx(ctx, key, cfg.CandidateTTL)
}

func LoadCandidateCache(
	ctx context.Context,
	rds *gzredis.Redis,
	key string,
	limit int,
) ([]Candidate, bool, error) {
	if rds == nil || strings.TrimSpace(key) == "" || limit <= 0 {
		return []Candidate{}, false, nil
	}

	pairs, err := rds.ZrevrangeWithScoresByFloatCtx(ctx, key, 0, int64(limit-1))
	if err != nil {
		return nil, false, err
	}
	if len(pairs) == 0 {
		return []Candidate{}, false, nil
	}

	candidates := make([]Candidate, 0, len(pairs))
	for _, pair := range pairs {
		id, err := strconv.ParseInt(pair.Key, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		candidates = append(candidates, Candidate{
			ContentID: id,
			Score:     float64(pair.Score) / candidateCacheScoreScale,
		})
	}
	if len(candidates) == 0 {
		return []Candidate{}, false, nil
	}
	return candidates, true, nil
}

const candidateCacheScoreScale = 1_000_000

func normalizeCacheSegment(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
