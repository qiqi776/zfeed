package recommend

import (
	"testing"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestMergeDedupsAndWeightsSources(t *testing.T) {
	got := Merge([]MergeInput{
		{
			Source: SourceHot,
			Weight: 1,
			IDs:    []int64{1001, 1002, 1003},
		},
		{
			Source: SourceNewContent,
			Weight: 0.5,
			IDs:    []int64{1003, 1004},
		},
	}, 10)

	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4", len(got))
	}
	if got[0].ContentID != 1001 {
		t.Fatalf("first content id = %d, want 1001", got[0].ContentID)
	}

	var merged Candidate
	for _, candidate := range got {
		if candidate.ContentID == 1003 {
			merged = candidate
			break
		}
	}
	if merged.ContentID == 0 {
		t.Fatal("merged candidate 1003 not found")
	}
	if merged.SourceRanks[SourceHot] != 3 || merged.SourceRanks[SourceNewContent] != 1 {
		t.Fatalf("source ranks = %#v, want hot=3 new=1", merged.SourceRanks)
	}
	if merged.SourceScores[SourceHot] == 0 || merged.SourceScores[SourceNewContent] == 0 {
		t.Fatalf("source scores not recorded: %#v", merged.SourceScores)
	}
}

func TestMergeUsesInputCandidateSourceScoresAndRanks(t *testing.T) {
	got := Merge([]MergeInput{
		{
			Source: SourceInterest,
			Weight: 0.25,
			Candidates: []Candidate{
				{
					ContentID: 2001,
					SourceScores: map[Source]float64{
						SourceInterest: 0.9,
					},
					SourceRanks: map[Source]int{
						SourceInterest: 7,
					},
				},
			},
		},
	}, 10)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	candidate := got[0]
	if candidate.SourceScores[SourceInterest] != 0.9 {
		t.Fatalf("interest source score = %v, want 0.9", candidate.SourceScores[SourceInterest])
	}
	if candidate.SourceRanks[SourceInterest] != 7 {
		t.Fatalf("interest source rank = %v, want 7", candidate.SourceRanks[SourceInterest])
	}
	if candidate.InterestScore != 0.9 {
		t.Fatalf("interest score = %v, want 0.9", candidate.InterestScore)
	}
	if candidate.Score < 0.224 || candidate.Score > 0.226 {
		t.Fatalf("merge score = %v, want 0.225", candidate.Score)
	}
}

func TestMergeLimit(t *testing.T) {
	got := Merge([]MergeInput{
		{
			Source: SourceHot,
			Weight: 1,
			IDs:    []int64{1001, 1002, 1003},
		},
	}, 2)

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ContentID != 1001 || got[1].ContentID != 1002 {
		t.Fatalf("ids = [%d %d], want [1001 1002]", got[0].ContentID, got[1].ContentID)
	}
}

func TestCandidateScoreFeaturesPreserveMergeScore(t *testing.T) {
	got := Merge([]MergeInput{
		{
			Source: SourceHot,
			Weight: 1,
			IDs:    []int64{1001},
		},
	}, 10)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Score != 1 {
		t.Fatalf("merge score = %f, want 1", got[0].Score)
	}

	candidate := got[0]
	candidate.HotScore = 0.9
	candidate.InterestScore = 0.8
	candidate.FreshnessScore = 0.7
	candidate.QualityScore = 0.6

	if candidate.Score != 1 {
		t.Fatalf("score changed after setting feature scores: %f", candidate.Score)
	}
}

func TestHotCandidatesFromPairsNormalizesScoreAndRanks(t *testing.T) {
	got := HotCandidatesFromPairs([]gzredis.FloatPair{
		{Key: "8302", Score: 100},
		{Key: "8301", Score: 80},
		{Key: "invalid", Score: 60},
	})

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ContentID != 8302 || got[1].ContentID != 8301 {
		t.Fatalf("ids = [%d %d], want [8302 8301]", got[0].ContentID, got[1].ContentID)
	}
	if got[0].HotScore != 1 {
		t.Fatalf("first hot score = %v, want 1", got[0].HotScore)
	}
	if got[1].HotScore != 0.8 {
		t.Fatalf("second hot score = %v, want 0.8", got[1].HotScore)
	}
	if got[0].SourceScores[SourceHot] != 1 || got[1].SourceScores[SourceHot] != 0.8 {
		t.Fatalf("hot source scores = %#v %#v", got[0].SourceScores, got[1].SourceScores)
	}
	if got[0].SourceRanks[SourceHot] != 1 || got[1].SourceRanks[SourceHot] != 2 {
		t.Fatalf("hot source ranks = %#v %#v", got[0].SourceRanks, got[1].SourceRanks)
	}
}

func TestDiversityRerankSpreadsAuthors(t *testing.T) {
	candidates := []Candidate{
		{ContentID: 1001, AuthorID: 10, ContentType: 10},
		{ContentID: 1002, AuthorID: 10, ContentType: 10},
		{ContentID: 1003, AuthorID: 20, ContentType: 20},
		{ContentID: 1004, AuthorID: 10, ContentType: 10},
	}

	got := DiversityRerank(candidates, contentconfig.RecommendDiversityConfig{
		Enabled:       true,
		AuthorWindow:  2,
		MaxSameAuthor: 1,
		TypeWindow:    3,
		MaxSameType:   2,
	})

	if len(got) != len(candidates) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(candidates))
	}
	if got[0].AuthorID == got[1].AuthorID {
		t.Fatalf("first two authors = [%d %d], want spread", got[0].AuthorID, got[1].AuthorID)
	}
	if got[0].ContentID != 1001 || got[1].ContentID != 1003 {
		t.Fatalf("first ids = [%d %d], want [1001 1003]", got[0].ContentID, got[1].ContentID)
	}
}

func TestDiversityRerankWithAdjustmentsRecordsRules(t *testing.T) {
	candidates := []Candidate{
		{ContentID: 1001, AuthorID: 10, ContentType: 10},
		{ContentID: 1002, AuthorID: 10, ContentType: 20},
		{ContentID: 1003, AuthorID: 20, ContentType: 10},
	}

	got, adjustments := DiversityRerankWithAdjustments(candidates, contentconfig.RecommendDiversityConfig{
		Enabled:       true,
		AuthorWindow:  2,
		MaxSameAuthor: 1,
		TypeWindow:    2,
		MaxSameType:   1,
	})

	if len(got) != len(candidates) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(candidates))
	}
	if adjustments[DiversityRuleAuthorWindow] == 0 {
		t.Fatalf("adjustments = %#v, want author window adjustment", adjustments)
	}
	if adjustments[DiversityRuleTypeWindow] == 0 {
		t.Fatalf("adjustments = %#v, want type window adjustment", adjustments)
	}
}

func TestDiversityRerankPromotesNewContentQuota(t *testing.T) {
	now := time.Now().Unix()
	old := time.Now().Add(-72 * time.Hour).Unix()
	candidates := []Candidate{
		{ContentID: 1001, PublishedAt: old},
		{ContentID: 1002, PublishedAt: old},
		{ContentID: 1003, PublishedAt: old},
		{ContentID: 1004, PublishedAt: old},
		{ContentID: 1005, PublishedAt: old},
		{ContentID: 2001, PublishedAt: now - 3600},
		{ContentID: 2002, PublishedAt: now - 2*3600},
	}

	got := DiversityRerank(candidates, contentconfig.RecommendDiversityConfig{
		Enabled:            true,
		NewContentTopN:     5,
		NewContentMinCount: 2,
	})

	if len(got) != len(candidates) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(candidates))
	}
	if countNewContent(got[:5], time.Unix(now, 0)) < 2 {
		t.Fatalf("top ids = %v, want at least two new contents in first 5", IDs(got[:5]))
	}
}

func TestApplyFeaturesDropsMissingRows(t *testing.T) {
	got := ApplyFeatures([]Candidate{
		{ContentID: 1001},
		{ContentID: 9999},
	}, map[int64]Candidate{
		1001: {ContentID: 1001, AuthorID: 42, ContentType: 10, PublishedAt: 123},
	})

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].AuthorID != 42 || got[0].ContentType != 10 || got[0].PublishedAt != 123 {
		t.Fatalf("unexpected feature application: %+v", got[0])
	}
}

func countNewContent(candidates []Candidate, now time.Time) int {
	count := 0
	for _, candidate := range candidates {
		if now.Unix()-candidate.PublishedAt < 24*3600 {
			count++
		}
	}
	return count
}
