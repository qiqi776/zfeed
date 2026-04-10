package hotrank

import (
	"math"
	"sort"
	"testing"
	"time"
)

func TestScoreWeights(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	formula := DefaultFormula()

	got := formula.Score(1, 1, 1, now, now)
	want := Round3(math.Log1p(1*1 + 1*3 + 1*4))
	if got != want {
		t.Fatalf("unexpected score, got=%v want=%v", got, want)
	}
}

func TestScoreTimeDecay(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	publishedAt := now.Add(-24 * time.Hour)
	formula := DefaultFormula()

	got := formula.Score(1, 1, 1, publishedAt, now)
	raw := math.Log1p(1*1 + 1*3 + 1*4)
	want := Round3(raw * 0.5)
	if got != want {
		t.Fatalf("unexpected decayed score, got=%v want=%v", got, want)
	}
}

func TestScoreFuturePublishedAtClamp(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	future := now.Add(2 * time.Hour)
	formula := DefaultFormula()

	got := formula.Score(3, 2, 1, future, now)
	want := formula.Score(3, 2, 1, now, now)
	if got != want {
		t.Fatalf("expected future published_at to be clamped, got=%v want=%v", got, want)
	}
}

func TestRound3(t *testing.T) {
	if got := Round3(1.23444); got != 1.234 {
		t.Fatalf("unexpected round result, got=%v want=1.234", got)
	}
	if got := Round3(1.23456); got != 1.235 {
		t.Fatalf("unexpected round result, got=%v want=1.235", got)
	}
}

func TestScoreWeightOverrideAndFallback(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	formula := Formula{
		Weights: Weights{
			Like:     2,
			Comment:  0,
			Favorite: -1,
		},
		HalfLifeHours: 24,
	}

	got := formula.Score(1, 1, 1, now, now)
	want := Round3(math.Log1p(1*2 + 1*3 + 1*4))
	if got != want {
		t.Fatalf("unexpected score with override/fallback, got=%v want=%v", got, want)
	}
}

func TestScoreNoDecayWhenHalfLifeNonPositive(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	publishedAt := now.Add(-72 * time.Hour)
	formula := Formula{
		Weights:       DefaultWeights(),
		HalfLifeHours: 0,
	}

	got := formula.Score(2, 0, 0, publishedAt, now)
	want := Round3(math.Log1p(2))
	if got != want {
		t.Fatalf("unexpected score with no decay, got=%v want=%v", got, want)
	}
}

func TestScoreRankingEvidence(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	formula := DefaultFormula()

	type sample struct {
		id       int64
		like     int64
		comment  int64
		favorite int64
		ageHours int
		score    float64
	}

	samples := []sample{
		{id: 1001, like: 40, comment: 10, favorite: 6, ageHours: 1},
		{id: 1002, like: 20, comment: 4, favorite: 2, ageHours: 2},
		{id: 1003, like: 12, comment: 2, favorite: 1, ageHours: 8},
		{id: 1004, like: 8, comment: 1, favorite: 0, ageHours: 20},
	}

	for i := range samples {
		publishedAt := now.Add(-time.Duration(samples[i].ageHours) * time.Hour)
		samples[i].score = formula.Score(samples[i].like, samples[i].comment, samples[i].favorite, publishedAt, now)
	}

	sort.Slice(samples, func(i, j int) bool {
		if samples[i].score == samples[j].score {
			return samples[i].id > samples[j].id
		}
		return samples[i].score > samples[j].score
	})

	gotOrder := []int64{samples[0].id, samples[1].id, samples[2].id, samples[3].id}
	wantOrder := []int64{1001, 1002, 1003, 1004}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("unexpected ranking order, got=%v want=%v", gotOrder, wantOrder)
		}
	}
}
