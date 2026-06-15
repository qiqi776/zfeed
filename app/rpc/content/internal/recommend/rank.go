package recommend

import (
	"math"
	"sort"

	contentconfig "zfeed/app/rpc/content/internal/config"
)

func CoarseRank(candidates []Candidate, cfg contentconfig.RecommendRankConfig) []Candidate {
	if len(candidates) == 0 {
		return []Candidate{}
	}
	cfg = NormalizeConfig(contentconfig.RecommendConfig{Rank: cfg}).Rank

	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ContentID <= 0 {
			continue
		}
		candidate.Score -= float64(candidate.SeenCount) * cfg.SeenPenalty
		result = append(result, candidate)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].ContentID > result[j].ContentID
		}
		return result[i].Score > result[j].Score
	})
	if len(result) > cfg.CoarseLimit {
		return result[:cfg.CoarseLimit]
	}
	return result
}

func FineRank(candidates []Candidate, cfg contentconfig.RecommendRankConfig) []Candidate {
	if len(candidates) == 0 {
		return []Candidate{}
	}
	cfg = NormalizeConfig(contentconfig.RecommendConfig{Rank: cfg}).Rank

	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ContentID <= 0 {
			continue
		}
		candidate.FinalScore = candidate.Score +
			cfg.AlphaHot*candidate.HotScore +
			cfg.BetaInterest*candidate.InterestScore +
			cfg.GammaFresh*candidate.FreshnessScore +
			cfg.DeltaQuality*candidate.QualityScore -
			float64(candidate.SeenCount)*cfg.SeenPenalty
		result = append(result, candidate)
	}
	result = filterRepeatedSeen(result, cfg.RepeatedSeenFilterN)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].FinalScore == result[j].FinalScore {
			return result[i].ContentID > result[j].ContentID
		}
		return result[i].FinalScore > result[j].FinalScore
	})
	return result
}

func filterRepeatedSeen(candidates []Candidate, threshold int) []Candidate {
	if len(candidates) <= 1 || threshold <= 0 {
		return candidates
	}

	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.SeenCount >= threshold {
			continue
		}
		result = append(result, candidate)
	}
	if len(result) == 0 {
		return candidates
	}
	return result
}

func InterestScore(userTags, contentTags map[string]float64) float64 {
	var dot float64
	var userNorm float64
	var contentNorm float64

	for tag, userWeight := range userTags {
		userNorm += userWeight * userWeight
		if contentWeight, ok := contentTags[tag]; ok {
			dot += userWeight * contentWeight
		}
	}
	for _, contentWeight := range contentTags {
		contentNorm += contentWeight * contentWeight
	}
	if userNorm == 0 || contentNorm == 0 {
		return 0
	}
	return dot / math.Sqrt(userNorm*contentNorm)
}

func rankIDs(scoreByID map[int64]float64, limit int) []int64 {
	return IDs(rankedCandidates(scoreByID, limit))
}

func rankedCandidates(scoreByID map[int64]float64, limit int) []Candidate {
	if limit <= 0 {
		limit = defaultInterestLimit
	}
	candidates := make([]Candidate, 0, len(scoreByID))
	for id, score := range scoreByID {
		if id <= 0 || score == 0 {
			continue
		}
		candidates = append(candidates, Candidate{ContentID: id, Score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].ContentID > candidates[j].ContentID
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}
