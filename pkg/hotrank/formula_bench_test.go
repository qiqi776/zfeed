package hotrank

import (
	"testing"
	"time"
)

var benchmarkHotScore float64

func BenchmarkFormulaScore(b *testing.B) {
	formula := DefaultFormula()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	publishedAt := now.Add(-6 * time.Hour)

	b.ReportAllocs()
	for b.Loop() {
		benchmarkHotScore = formula.Score(42, 8, 13, publishedAt, now)
	}
}

func BenchmarkFormulaScoreRankingBatch(b *testing.B) {
	formula := DefaultFormula()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	samples := []struct {
		like     int64
		comment  int64
		favorite int64
		age      time.Duration
	}{
		{like: 40, comment: 10, favorite: 6, age: time.Hour},
		{like: 20, comment: 4, favorite: 2, age: 2 * time.Hour},
		{like: 12, comment: 2, favorite: 1, age: 8 * time.Hour},
		{like: 8, comment: 1, favorite: 0, age: 20 * time.Hour},
	}

	b.ReportAllocs()
	for b.Loop() {
		var total float64
		for _, sample := range samples {
			total += formula.Score(sample.like, sample.comment, sample.favorite, now.Add(-sample.age), now)
		}
		benchmarkHotScore = total
	}
}
