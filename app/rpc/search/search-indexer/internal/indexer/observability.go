package indexer

import (
	"time"

	"github.com/zeromicro/go-zero/core/metric"
)

const (
	indexEntityContent = "content"
	indexEntityUser    = "user"

	indexOperationIndex      = "index"
	indexOperationDelete     = "delete"
	indexOperationBulkIndex  = "bulk_index"
	indexOperationBulkDelete = "bulk_delete"

	indexResultSuccess = "success"
	indexResultError   = "error"
)

var (
	metricSearchIndexerWriteTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "indexer_write",
		Name:      "total",
		Help:      "Search indexer write operation count.",
		Labels:    []string{"entity", "operation", "result"},
	})

	metricSearchIndexerWriteDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "indexer_write",
		Name:      "duration_ms",
		Help:      "Search indexer write operation duration in milliseconds.",
		Labels:    []string{"entity", "operation", "result"},
		Buckets:   []float64{1, 3, 5, 10, 25, 50, 100, 250, 500, 1000, 3000},
	})
)

func observeWrite(entity string, operation string, start time.Time, err error) {
	result := indexResultSuccess
	if err != nil {
		result = indexResultError
	}
	metricSearchIndexerWriteTotal.Inc(entity, operation, result)
	metricSearchIndexerWriteDuration.Observe(time.Since(start).Milliseconds(), entity, operation, result)
}
