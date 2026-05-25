package consumer

import (
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/metric"
)

const (
	consumeResultSuccess    = "success"
	consumeResultError      = "error"
	consumeResultParseError = "parse_error"
	unknownLabel            = "unknown"
)

var (
	metricSearchIndexerConsumeTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "indexer_consume",
		Name:      "total",
		Help:      "Search indexer Canal consume count.",
		Labels:    []string{"table", "event_type", "result"},
	})

	metricSearchIndexerConsumeDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "indexer_consume",
		Name:      "duration_ms",
		Help:      "Search indexer Canal consume duration in milliseconds.",
		Labels:    []string{"table", "event_type", "result"},
		Buckets:   []float64{1, 3, 5, 10, 25, 50, 100, 250, 500, 1000, 3000},
	})

	metricSearchIndexerConsumeLag = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "indexer_consume",
		Name:      "lag_ms",
		Help:      "Search indexer Canal event lag in milliseconds.",
		Labels:    []string{"table", "event_type", "result"},
		Buckets:   []float64{0, 100, 500, 1000, 3000, 5000, 10000, 30000, 60000},
	})
)

func observeCanalConsume(table string, eventType string, eventTS int64, start time.Time, result string) {
	table = normalizeMetricLabel(table)
	eventType = normalizeMetricLabel(eventType)
	metricSearchIndexerConsumeTotal.Inc(table, eventType, result)
	metricSearchIndexerConsumeDuration.Observe(time.Since(start).Milliseconds(), table, eventType, result)
	metricSearchIndexerConsumeLag.Observe(eventLagMs(eventTS), table, eventType, result)
}

func eventLagMs(eventTS int64) int64 {
	lag := time.Since(canalTimestampToTime(eventTS)).Milliseconds()
	if lag < 0 {
		return 0
	}
	return lag
}

func normalizeMetricLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return unknownLabel
	}
	return value
}
