package querynorm

import "testing"

var benchmarkQuery Query

func BenchmarkDefaultNormalizerNormalize(b *testing.B) {
	normalizer := NewDefaultNormalizer()
	queries := []string{
		"  Bench   Article  ",
		"bench_user_10001",
		"13800138000",
		"feed recommendation search keyword",
		"   ",
	}

	b.ReportAllocs()
	for b.Loop() {
		for _, query := range queries {
			benchmarkQuery = normalizer.Normalize(query)
		}
	}
}
