package recommend

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/pkg/redisx"
)

func RecallInterest(
	ctx context.Context,
	rds *redis.Redis,
	userID int64,
	cfg contentconfig.RecommendInterestConfig,
) ([]int64, error) {
	candidates, err := RecallInterestCandidates(ctx, rds, userID, cfg)
	if err != nil {
		return nil, err
	}
	return IDs(candidates), nil
}

func RecallInterestCandidates(
	ctx context.Context,
	rds *redis.Redis,
	userID int64,
	cfg contentconfig.RecommendInterestConfig,
) ([]Candidate, error) {
	if rds == nil || userID <= 0 || !cfg.Enabled {
		return []Candidate{}, nil
	}
	cfg = NormalizeConfig(contentconfig.RecommendConfig{Interest: cfg}).Interest

	profile, err := LoadUserProfile(ctx, rds, userID, cfg)
	if err != nil || len(profile) == 0 {
		return []Candidate{}, err
	}

	scoreByID := make(map[int64]float64)
	rankByID := make(map[int64]int)
	tagPairs := make([]interestTagPairs, 0, len(profile))
	var maxScore float64
	for tag, profileWeight := range profile {
		pairs, err := redisx.ZRangeRevWithScoresByFloatCtx(
			ctx,
			rds,
			redisconsts.BuildRecommendTagIndexKey(tag),
			0,
			int64(cfg.Limit-1),
		)
		if err != nil {
			return nil, err
		}
		tagPairs = append(tagPairs, interestTagPairs{
			profileWeight: profileWeight,
			pairs:         pairs,
		})
		maxScore = max(maxScore, maxFloatPairScore(pairs))
	}
	for _, tagPair := range tagPairs {
		for rank, pair := range tagPair.pairs {
			id, err := strconv.ParseInt(pair.Key, 10, 64)
			if err != nil || id <= 0 {
				continue
			}
			scoreByID[id] += tagPair.profileWeight * normalizeTagIndexScore(pair.Score, maxScore)
			rankByID[id] = minPositive(rankByID[id], rank+1)
		}
	}

	return rankInterestCandidates(scoreByID, rankByID, cfg.Limit), nil
}

type interestTagPairs struct {
	profileWeight float64
	pairs         []redis.FloatPair
}

func maxFloatPairScore(pairs []redis.FloatPair) float64 {
	var maxScore float64
	for _, pair := range pairs {
		if pair.Score > maxScore {
			maxScore = pair.Score
		}
	}
	return maxScore
}

func normalizeTagIndexScore(score, maxScore float64) float64 {
	if score <= 0 || maxScore <= 0 {
		return 0
	}
	return score / maxScore
}

func rankInterestCandidates(scoreByID map[int64]float64, rankByID map[int64]int, limit int) []Candidate {
	candidates := rankedCandidates(scoreByID, limit)
	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.InterestScore = candidate.Score
		candidate.SourceScores = map[Source]float64{
			SourceInterest: candidate.Score,
		}
		if sourceRank := rankByID[candidate.ContentID]; sourceRank > 0 {
			candidate.SourceRanks = map[Source]int{
				SourceInterest: sourceRank,
			}
		}
		result = append(result, candidate)
	}
	return result
}
