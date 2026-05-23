package logic

import (
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/metric"

	"zfeed/app/rpc/search/search-rpc/internal/querynorm"
	"zfeed/app/rpc/search/search-rpc/internal/repositories"
	"zfeed/app/rpc/search/search-rpc/internal/svc"
)

const (
	searchEntityUsers    = "users"
	searchEntityContents = "contents"

	searchCacheDisabled = "disabled"
	searchCacheBypass   = "bypass"
	searchCacheHit      = "hit"
	searchCacheMiss     = "miss"
	searchCacheFallback = "fallback"
	searchCacheError    = "error"

	defaultSearchSlowThreshold = 200 * time.Millisecond
)

var (
	metricSearchRequestDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "request",
		Name:      "duration_ms",
		Help:      "Search request duration in milliseconds.",
		Labels:    []string{"entity", "backend", "query_path", "db_fallback", "cache_status", "result"},
		Buckets:   []float64{1, 3, 5, 10, 25, 50, 100, 200, 500, 1000, 2000},
	})

	metricSearchRequestTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "request",
		Name:      "total",
		Help:      "Search request count.",
		Labels:    []string{"entity", "backend", "query_path", "db_fallback", "cache_status", "result"},
	})

	metricSearchResultCount = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "result",
		Name:      "count",
		Help:      "Search result count.",
		Labels:    []string{"entity", "backend", "query_path", "db_fallback", "cache_status"},
		Buckets:   []float64{0, 1, 3, 5, 10, 20, 50},
	})

	metricSearchSlowTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "request",
		Name:      "slow_total",
		Help:      "Search slow request count.",
		Labels:    []string{"entity", "backend", "query_path", "db_fallback", "cache_status"},
	})

	metricSearchCacheTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed_search",
		Subsystem: "cache",
		Name:      "total",
		Help:      "Search cache event count.",
		Labels:    []string{"layer", "entity", "mode", "status"},
	})
)

type searchObservation struct {
	entity            string
	query             querynorm.Query
	cursor            int64
	pageSize          int
	resultCount       int
	hasMore           bool
	mode              string
	pageTokenProvided bool
	snapshotID        string
	snapshotStatus    string
	err               error
	start             time.Time
	meta              repositories.SearchMeta
	cacheStatus       string
	configuredBackend string
	effectiveBackend  string
	svcCtx            *svc.ServiceContext
}

func observeSearch(logger logx.Logger, obs searchObservation) {
	elapsed := time.Since(obs.start)
	result := searchResultLabel(obs.resultCount, obs.err)
	dbFallback := strconv.FormatBool(obs.meta.DBFallback)
	labels := []string{
		obs.entity,
		obs.effectiveBackend,
		normalizeQueryPath(obs.meta.QueryPath),
		dbFallback,
		obs.cacheStatus,
		result,
	}

	metricSearchRequestDuration.Observe(elapsed.Milliseconds(), labels...)
	metricSearchRequestTotal.Inc(labels...)
	if obs.err == nil {
		metricSearchResultCount.Observe(
			int64(obs.resultCount),
			obs.entity,
			obs.effectiveBackend,
			normalizeQueryPath(obs.meta.QueryPath),
			dbFallback,
			obs.cacheStatus,
		)
	}
	if elapsed >= defaultSearchSlowThreshold {
		metricSearchSlowTotal.Inc(
			obs.entity,
			obs.effectiveBackend,
			normalizeQueryPath(obs.meta.QueryPath),
			dbFallback,
			obs.cacheStatus,
		)
	}

	fields := []logx.LogField{
		logx.Field("entity", obs.entity),
		logx.Field("keyword", obs.query.LogValue),
		logx.Field("query", obs.query.LogValue),
		logx.Field("query_hash", obs.query.Hash),
		logx.Field("cursor", obs.cursor),
		logx.Field("page_size", obs.pageSize),
		logx.Field("result_count", obs.resultCount),
		logx.Field("has_more", obs.hasMore),
		logx.Field("mode", obs.mode),
		logx.Field("page_token_provided", obs.pageTokenProvided),
		logx.Field("snapshot_id", obs.snapshotID),
		logx.Field("snapshot_status", obs.snapshotStatus),
		logx.Field("result", result),
		logx.Field("configured_backend", obs.configuredBackend),
		logx.Field("effective_backend", obs.effectiveBackend),
		logx.Field("query_path", normalizeQueryPath(obs.meta.QueryPath)),
		logx.Field("db_fallback", obs.meta.DBFallback),
		logx.Field("cache_status", obs.cacheStatus),
		logx.Field("elapsed_ms", elapsed.Milliseconds()),
		logx.Field("search_cache_enabled", obs.svcCtx != nil && obs.svcCtx.Config.SearchCacheEnabled),
		logx.Field("search_snapshot_enabled", obs.svcCtx != nil && obs.svcCtx.Config.SearchSnapshotEnabled),
		logx.Field("search_hybrid_rank_enabled", obs.svcCtx != nil && obs.svcCtx.Config.SearchHybridRankEnabled),
	}

	if logger != nil {
		logger.Infow("search request observed", fields...)
		if elapsed >= defaultSearchSlowThreshold {
			logger.Sloww("search request slow", fields...)
		}
	}
}

func searchResultLabel(resultCount int, err error) string {
	if err != nil {
		return "error"
	}
	if resultCount == 0 {
		return "zero"
	}
	return "nonzero"
}

func cacheStatus(svcCtx *svc.ServiceContext) string {
	if svcCtx != nil && svcCtx.Config.SearchCacheEnabled {
		return searchCacheBypass
	}
	return searchCacheDisabled
}

func observeSearchCache(layer string, entity string, mode string, status string) {
	metricSearchCacheTotal.Inc(layer, entity, mode, status)
}

func normalizeQueryPath(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
