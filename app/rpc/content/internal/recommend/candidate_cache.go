package recommend

import (
	"context"
	"encoding/json"
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
	if err := saveCandidateCacheSources(ctx, rds, cfg, key, candidates); err != nil {
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
	if err := attachCandidateCacheSources(ctx, rds, key, candidates); err != nil {
		return nil, false, err
	}
	return candidates, true, nil
}

const candidateCacheScoreScale = 1_000_000
const candidateCacheSourceSuffix = ":source"
const candidateCacheDetailSuffix = ":detail"

type candidateCacheDetail struct {
	Scores map[string]float64 `json:"scores,omitempty"`
	Ranks  map[string]int     `json:"ranks,omitempty"`
}

func saveCandidateCacheSources(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	key string,
	candidates []Candidate,
) error {
	sourceKey := buildCandidateCacheSourceKey(key)
	detailKey := buildCandidateCacheDetailKey(key)
	wrote := false
	wroteDetail := false
	for _, candidate := range candidates {
		if candidate.ContentID <= 0 {
			continue
		}

		source := PrimarySource(candidate)
		if source == "" {
			detailWritten, err := saveCandidateCacheDetail(ctx, rds, detailKey, candidate)
			if err != nil {
				return err
			}
			if detailWritten {
				wroteDetail = true
			}
			continue
		}
		if err := rds.HsetCtx(
			ctx,
			sourceKey,
			strconv.FormatInt(candidate.ContentID, 10),
			string(source),
		); err != nil {
			return err
		}
		wrote = true
		detailWritten, err := saveCandidateCacheDetail(ctx, rds, detailKey, candidate)
		if err != nil {
			return err
		}
		if detailWritten {
			wroteDetail = true
		}
	}
	if !wrote {
		if !wroteDetail {
			return nil
		}
		if err := rds.ExpireCtx(ctx, detailKey, cfg.CandidateTTL); err != nil {
			return err
		}
		return nil
	}
	if err := rds.ExpireCtx(ctx, sourceKey, cfg.CandidateTTL); err != nil {
		return err
	}
	if wroteDetail {
		return rds.ExpireCtx(ctx, detailKey, cfg.CandidateTTL)
	}
	return nil
}

func saveCandidateCacheDetail(
	ctx context.Context,
	rds *gzredis.Redis,
	detailKey string,
	candidate Candidate,
) (bool, error) {
	detail, ok := buildCandidateCacheDetail(candidate)
	if !ok {
		return false, nil
	}
	if err := rds.HsetCtx(
		ctx,
		detailKey,
		strconv.FormatInt(candidate.ContentID, 10),
		detail,
	); err != nil {
		return false, err
	}
	return true, nil
}

func attachCandidateCacheSources(
	ctx context.Context,
	rds *gzredis.Redis,
	key string,
	candidates []Candidate,
) error {
	rawSources, err := rds.HgetallCtx(ctx, buildCandidateCacheSourceKey(key))
	if err != nil {
		return err
	}
	rawDetails, err := rds.HgetallCtx(ctx, buildCandidateCacheDetailKey(key))
	if err != nil {
		return err
	}

	for i := range candidates {
		contentID := strconv.FormatInt(candidates[i].ContentID, 10)
		if detail, ok := parseCandidateCacheDetail(rawDetails[contentID]); ok {
			if len(detail.Scores) > 0 {
				candidates[i].SourceScores = make(map[Source]float64, len(detail.Scores))
				for rawSource, score := range detail.Scores {
					source := normalizeSource(rawSource)
					if source == "" || score <= 0 {
						continue
					}
					candidates[i].SourceScores[source] = score
				}
			}
			if len(detail.Ranks) > 0 {
				candidates[i].SourceRanks = make(map[Source]int, len(detail.Ranks))
				for rawSource, rank := range detail.Ranks {
					source := normalizeSource(rawSource)
					if source == "" || rank <= 0 {
						continue
					}
					candidates[i].SourceRanks[source] = rank
				}
			}
			if len(candidates[i].SourceScores) > 0 || len(candidates[i].SourceRanks) > 0 {
				continue
			}
		}

		source := normalizeSource(rawSources[contentID])
		if source == "" {
			continue
		}
		candidates[i].SourceScores = map[Source]float64{
			source: 1,
		}
		candidates[i].SourceRanks = map[Source]int{
			source: 1,
		}
	}
	return nil
}

func buildCandidateCacheSourceKey(key string) string {
	return key + candidateCacheSourceSuffix
}

func buildCandidateCacheDetailKey(key string) string {
	return key + candidateCacheDetailSuffix
}

func buildCandidateCacheDetail(candidate Candidate) (string, bool) {
	detail := candidateCacheDetail{}
	if len(candidate.SourceScores) > 0 {
		detail.Scores = make(map[string]float64, len(candidate.SourceScores))
		for source, score := range candidate.SourceScores {
			normalized := normalizeSource(string(source))
			if normalized == "" || score <= 0 {
				continue
			}
			detail.Scores[string(normalized)] = score
		}
	}
	if len(candidate.SourceRanks) > 0 {
		detail.Ranks = make(map[string]int, len(candidate.SourceRanks))
		for source, rank := range candidate.SourceRanks {
			normalized := normalizeSource(string(source))
			if normalized == "" || rank <= 0 {
				continue
			}
			detail.Ranks[string(normalized)] = rank
		}
	}
	if len(detail.Scores) == 0 && len(detail.Ranks) == 0 {
		return "", false
	}
	raw, err := json.Marshal(detail)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func parseCandidateCacheDetail(raw string) (candidateCacheDetail, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return candidateCacheDetail{}, false
	}

	var detail candidateCacheDetail
	if err := json.Unmarshal([]byte(raw), &detail); err != nil {
		return candidateCacheDetail{}, false
	}
	return detail, true
}

func normalizeCacheSegment(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
