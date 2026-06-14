package recommend

import (
	"math"
	"testing"

	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestInterestScoreCosineSimilarity(t *testing.T) {
	got := InterestScore(
		map[string]float64{"go": 3, "redis": 4},
		map[string]float64{"go": 3, "redis": 4},
	)
	if math.Abs(got-1) > 0.000001 {
		t.Fatalf("InterestScore() = %f, want 1", got)
	}
}

func TestInterestScoreHandlesEmptyVectors(t *testing.T) {
	if got := InterestScore(nil, map[string]float64{"go": 1}); got != 0 {
		t.Fatalf("InterestScore(empty user) = %f, want 0", got)
	}
	if got := InterestScore(map[string]float64{"go": 1}, nil); got != 0 {
		t.Fatalf("InterestScore(empty content) = %f, want 0", got)
	}
}

func TestFineRankUsesConfigWeights(t *testing.T) {
	candidates := []Candidate{
		{ContentID: 1001, HotScore: 0.9, InterestScore: 0.1, FreshnessScore: 0.1, QualityScore: 0.1},
		{ContentID: 1002, HotScore: 0.1, InterestScore: 0.9, FreshnessScore: 0.1, QualityScore: 0.1},
	}

	got := FineRank(candidates, contentconfig.RecommendRankConfig{
		AlphaHot:     0.1,
		BetaInterest: 0.8,
		GammaFresh:   0.05,
		DeltaQuality: 0.05,
	})
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ContentID != 1002 {
		t.Fatalf("top content id = %d, want interest-heavy candidate 1002", got[0].ContentID)
	}
}

func TestFineRankAppliesSeenPenalty(t *testing.T) {
	candidates := []Candidate{
		{ContentID: 1001, HotScore: 0.9, SeenCount: 2},
		{ContentID: 1002, HotScore: 0.8},
	}

	got := FineRank(candidates, contentconfig.RecommendRankConfig{
		AlphaHot:    1,
		SeenPenalty: 0.1,
	})
	if got[0].ContentID != 1002 {
		t.Fatalf("top content id = %d, want unseen candidate 1002", got[0].ContentID)
	}
}
