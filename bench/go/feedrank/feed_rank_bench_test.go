package feedrank

import "testing"

func TestMergeRankAndFilterSeenCandidates(t *testing.T) {
	inputs := []recallInput{
		{
			source: "hot",
			weight: 0.55,
			candidates: []candidate{
				{contentID: 9001, sourceScores: map[string]float64{"hot": 1.00}, sourceRanks: map[string]int{"hot": 1}},
				{contentID: 9002, sourceScores: map[string]float64{"hot": 0.90}, sourceRanks: map[string]int{"hot": 2}},
				{contentID: 9003, sourceScores: map[string]float64{"hot": 0.80}, sourceRanks: map[string]int{"hot": 3}},
			},
		},
		{
			source: "interest",
			weight: 0.45,
			candidates: []candidate{
				{contentID: 9002, sourceScores: map[string]float64{"interest": 1.00}, sourceRanks: map[string]int{"interest": 1}},
				{contentID: 9004, sourceScores: map[string]float64{"interest": 0.95}, sourceRanks: map[string]int{"interest": 2}},
			},
		},
	}
	features := map[int64]candidateFeature{
		9001: {qualityScore: 0.1, freshnessScore: 0.3},
		9002: {qualityScore: 0.8, freshnessScore: 0.9},
		9003: {qualityScore: 0.2, freshnessScore: 0.4},
		9004: {qualityScore: 0.6, freshnessScore: 0.7},
	}
	seenCounts := map[int64]int{9001: 3, 9002: 1}

	got := rankFeedCandidates(inputs, features, seenCounts, rankConfig{
		limit:              10,
		hotWeight:          0.45,
		interestWeight:     0.35,
		freshnessWeight:    0.15,
		qualityWeight:      0.05,
		seenPenalty:        0.25,
		repeatedSeenFilter: 2,
	})

	if len(got) != 3 {
		t.Fatalf("ranked candidates = %d, want 3 after filtering one repeated seen item: %+v", len(got), got)
	}
	if got[0].contentID != 9002 {
		t.Fatalf("top content id = %d, want merged hot+interest candidate 9002: %+v", got[0].contentID, got)
	}
	if got[0].sourceRanks["hot"] != 2 || got[0].sourceRanks["interest"] != 1 {
		t.Fatalf("merged source ranks = %#v, want hot=2 interest=1", got[0].sourceRanks)
	}
	for _, candidate := range got {
		if candidate.contentID == 9001 {
			t.Fatalf("content 9001 should be filtered by repeated seen count: %+v", got)
		}
	}
}

var (
	benchmarkRankedCandidates []candidate
	benchmarkCandidateIDs     []int64
)

func BenchmarkFeedCandidateMergeRankSeenFilter(b *testing.B) {
	inputs, features, seenCounts := buildFixture(400, 120)
	cfg := rankConfig{
		limit:              300,
		hotWeight:          0.45,
		interestWeight:     0.35,
		freshnessWeight:    0.15,
		qualityWeight:      0.05,
		seenPenalty:        0.25,
		repeatedSeenFilter: 3,
	}

	b.ReportAllocs()
	for b.Loop() {
		benchmarkRankedCandidates = rankFeedCandidates(inputs, features, seenCounts, cfg)
		if len(benchmarkRankedCandidates) == 0 {
			b.Fatal("ranked candidates is empty")
		}
	}
}

func BenchmarkFeedCandidateIDCollect(b *testing.B) {
	inputs, features, seenCounts := buildFixture(400, 120)
	ranked := rankFeedCandidates(inputs, features, seenCounts, rankConfig{
		limit:              300,
		hotWeight:          0.45,
		interestWeight:     0.35,
		freshnessWeight:    0.15,
		qualityWeight:      0.05,
		seenPenalty:        0.25,
		repeatedSeenFilter: 3,
	})

	b.ReportAllocs()
	for b.Loop() {
		benchmarkCandidateIDs = candidateIDs(ranked)
		if len(benchmarkCandidateIDs) != len(ranked) {
			b.Fatalf("candidate ids = %d, want %d", len(benchmarkCandidateIDs), len(ranked))
		}
	}
}
