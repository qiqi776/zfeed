package recommend

import (
	"testing"

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
