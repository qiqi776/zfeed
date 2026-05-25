package backend

import "github.com/zeromicro/go-zero/core/metric"

var (
	metricSearchCompareOverlapRatio = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "compare",
		Name:      "overlap_ratio",
		Help:      "Search shadow compare topN overlap ratio in basis points.",
		Labels:    []string{"entity", "primary_backend", "shadow_backend", "mode", "result"},
		Buckets:   []float64{0, 2500, 5000, 7000, 9000, 10000},
	})

	metricSearchEngineFallbackTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "engine",
		Name:      "fallback_total",
		Help:      "Search engine fallback count.",
		Labels:    []string{"entity", "reason"},
	})
)

func observeSearchCompare(entity string, primaryBackend string, shadowBackend string, mode string, result string, overlapRatio float64) {
	if overlapRatio < 0 {
		overlapRatio = 0
	}
	if overlapRatio > 1 {
		overlapRatio = 1
	}
	metricSearchCompareOverlapRatio.Observe(int64(overlapRatio*10000), entity, primaryBackend, shadowBackend, mode, result)
}

func observeEngineFallback(entity string, reason string) {
	metricSearchEngineFallbackTotal.Inc(entity, reason)
}
