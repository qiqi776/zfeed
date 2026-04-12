package orm

import "github.com/zeromicro/go-zero/core/metric"

const dbNamespace = "zfeed_db"

var (
	metricStatementDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: dbNamespace,
		Subsystem: "statement",
		Name:      "duration_ms",
		Help:      "Database statement duration in milliseconds.",
		Labels:    []string{"service", "operation", "table"},
		Buckets:   []float64{1, 3, 5, 10, 25, 50, 100, 250, 500, 1000, 2000},
	})

	metricStatementTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: dbNamespace,
		Subsystem: "statement",
		Name:      "total",
		Help:      "Database statement execution count.",
		Labels:    []string{"service", "operation", "table", "result"},
	})

	metricStatementSlowTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: dbNamespace,
		Subsystem: "statement",
		Name:      "slow_total",
		Help:      "Database slow statement count.",
		Labels:    []string{"service", "operation", "table"},
	})
)
