package feedlogic

import (
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/metric"
)

const (
	recommendMetricUnknownLabel = "unknown"

	recommendVariantControl = "control"

	recommendModeHot          = "hot"
	recommendModePersonalized = "personalized"
	recommendModeSnapshot     = "snapshot"
	recommendModeColdStart    = "cold_start"

	recommendResultSuccess  = "success"
	recommendResultError    = "error"
	recommendResultEmpty    = "empty"
	recommendResultFallback = "fallback"

	recommendStageTotal             = "total"
	recommendStageSnapshotLookup    = "snapshot_lookup"
	recommendStageRecall            = "recall"
	recommendStageFeatureLoad       = "feature_load"
	recommendStageCoarseRank        = "coarse_rank"
	recommendStageFineRank          = "fine_rank"
	recommendStageRerank            = "rerank"
	recommendStageBuildItems        = "build_items"
	recommendStageSnapshotSave      = "snapshot_save"
	recommendRerankRuleAuthorWindow = "author_window"
	recommendRerankRuleTypeWindow   = "type_window"

	recommendErrorStageCandidateCache = "candidate_cache"
	recommendErrorStageRecall         = "recall"
	recommendErrorStageFeatureLoad    = "feature_load"
	recommendErrorStageSeenLoad       = "seen_load"
	recommendErrorStageSnapshotSave   = "snapshot_save"
	recommendErrorStageSnapshotRead   = "snapshot_read"
	recommendErrorStageBuildItems     = "build_items"
	recommendErrorStageSeenWrite      = "seen_write"
	recommendErrorStageProfileUpdate  = "profile_update"

	recommendRecallSourceHot        = "hot"
	recommendRecallSourceNewContent = "new_content"
	recommendRecallSourceInterest   = "interest"

	recommendFallbackReasonDisabled         = "disabled"
	recommendFallbackReasonSnapshotMiss     = "snapshot_miss"
	recommendFallbackReasonSnapshotError    = "snapshot_error"
	recommendFallbackReasonEmptyRecall      = "empty_recall"
	recommendFallbackReasonEnhancementError = "enhancement_error"
	recommendFallbackReasonHotError         = "hot_error"
	recommendFallbackReasonBuildError       = "build_error"
	recommendFallbackReasonColdStart        = "cold_start"

	recommendSnapshotKindHot          = "hot"
	recommendSnapshotKindPersonalized = "personalized"

	recommendSnapshotResultHit     = "hit"
	recommendSnapshotResultMiss    = "miss"
	recommendSnapshotResultError   = "error"
	recommendSnapshotResultSaved   = "saved"
	recommendSnapshotResultSkipped = "skipped"

	recommendProfileResultDisabled = "disabled"
	recommendProfileResultSkipped  = "skipped"
	recommendProfileResultMiss     = "miss"
	recommendProfileResultHit      = "hit"
	recommendProfileResultError    = "error"
)

var (
	recommendRequestMetricLabels       = []string{"mode", "variant", "result"}
	recommendStageDurationMetricLabels = []string{"stage", "variant"}
	recommendRecallItemsMetricLabels   = []string{"source", "variant"}
	recommendFallbackMetricLabels      = []string{"reason"}
	recommendSnapshotMetricLabels      = []string{"kind", "result"}
	recommendRerankAdjustMetricLabels  = []string{"rule", "variant"}
	recommendErrorMetricLabels         = []string{"stage", "variant"}
	recommendProfileMetricLabels       = []string{"result"}
	recommendTrackMetricLabels         = []string{"event_type", "result"}

	metricRecommendRequestsTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_requests",
		Name:      "total",
		Help:      "Recommendation request count.",
		Labels:    recommendRequestMetricLabels,
	})

	metricRecommendStageDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_stage_duration",
		Name:      "seconds",
		Help:      "Recommendation stage duration in seconds.",
		Labels:    recommendStageDurationMetricLabels,
		Buckets:   []float64{0.001, 0.003, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	})

	metricRecommendRecallItemsTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_recall_items",
		Name:      "total",
		Help:      "Recommendation recalled item count.",
		Labels:    recommendRecallItemsMetricLabels,
	})

	metricRecommendFallbackTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_fallback",
		Name:      "total",
		Help:      "Recommendation fallback count.",
		Labels:    recommendFallbackMetricLabels,
	})

	metricRecommendSnapshotTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_snapshot",
		Name:      "total",
		Help:      "Recommendation snapshot event count.",
		Labels:    recommendSnapshotMetricLabels,
	})

	metricRecommendRerankAdjustTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_rerank_adjust",
		Name:      "total",
		Help:      "Recommendation rerank adjustment count.",
		Labels:    recommendRerankAdjustMetricLabels,
	})

	metricRecommendErrorTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_error",
		Name:      "total",
		Help:      "Recommendation enhancement error count by stage.",
		Labels:    recommendErrorMetricLabels,
	})

	metricRecommendProfileTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_profile",
		Name:      "total",
		Help:      "Recommendation profile lookup count.",
		Labels:    recommendProfileMetricLabels,
	})

	metricRecommendTrackEmitTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_track_emit",
		Name:      "total",
		Help:      "Recommendation track emission count.",
		Labels:    recommendTrackMetricLabels,
	})

	recordRecommendRequestMetric       = recordRecommendRequest
	recordRecommendStageDurationMetric = recordRecommendStageDuration
	recordRecommendRecallItemsMetric   = recordRecommendRecallItems
	recordRecommendFallbackMetric      = recordRecommendFallback
	recordRecommendSnapshotMetric      = recordRecommendSnapshot
	recordRecommendRerankAdjustMetric  = recordRecommendRerankAdjust
	recordRecommendErrorMetric         = recordRecommendError
	recordRecommendProfileMetric       = recordRecommendProfile
	recordRecommendTrackEmitMetric     = recordRecommendTrackEmit
)

func recordRecommendRequest(mode, variant, result string) {
	if metricRecommendRequestsTotal == nil {
		return
	}
	metricRecommendRequestsTotal.Inc(
		normalizeRecommendModeLabel(mode),
		normalizeRecommendVariantLabel(variant),
		normalizeRecommendResultLabel(result),
	)
}

func recordRecommendStageDuration(stage, variant string, elapsed time.Duration) {
	if metricRecommendStageDuration == nil {
		return
	}
	if elapsed < 0 {
		elapsed = 0
	}
	metricRecommendStageDuration.ObserveFloat(
		elapsed.Seconds(),
		normalizeRecommendStageLabel(stage),
		normalizeRecommendVariantLabel(variant),
	)
}

func recordRecommendRecallItems(source, variant string, count int) {
	if metricRecommendRecallItemsTotal == nil {
		return
	}
	if count < 0 {
		count = 0
	}
	metricRecommendRecallItemsTotal.Add(
		float64(count),
		normalizeRecommendRecallSourceLabel(source),
		normalizeRecommendVariantLabel(variant),
	)
}

func recordRecommendFallback(reason string) {
	if metricRecommendFallbackTotal == nil {
		return
	}
	metricRecommendFallbackTotal.Inc(normalizeRecommendFallbackReasonLabel(reason))
}

func recordRecommendSnapshot(kind, result string) {
	if metricRecommendSnapshotTotal == nil {
		return
	}
	metricRecommendSnapshotTotal.Inc(
		normalizeRecommendSnapshotKindLabel(kind),
		normalizeRecommendSnapshotResultLabel(result),
	)
}

func recordRecommendRerankAdjust(rule, variant string, count int) {
	if metricRecommendRerankAdjustTotal == nil {
		return
	}
	if count < 0 {
		count = 0
	}
	metricRecommendRerankAdjustTotal.Add(
		float64(count),
		normalizeRecommendRerankRuleLabel(rule),
		normalizeRecommendVariantLabel(variant),
	)
}

func recordRecommendError(stage, variant string) {
	if metricRecommendErrorTotal == nil {
		return
	}
	metricRecommendErrorTotal.Inc(
		normalizeRecommendErrorStageLabel(stage),
		normalizeRecommendVariantLabel(variant),
	)
}

func recordRecommendProfile(result string) {
	if metricRecommendProfileTotal == nil {
		return
	}
	metricRecommendProfileTotal.Inc(normalizeRecommendProfileResultLabel(result))
}

func recordRecommendTrackEmit(eventType, result string) {
	if metricRecommendTrackEmitTotal == nil {
		return
	}
	metricRecommendTrackEmitTotal.Inc(
		normalizeRecommendTrackEventTypeLabel(eventType),
		normalizeRecommendTrackEmitResultLabel(result),
	)
}

func normalizeRecommendModeLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendModeHot,
		recommendModePersonalized,
		recommendModeSnapshot,
		recommendModeColdStart:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendVariantLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case "a", "b", recommendVariantControl:
		return canonicalRecommendMetricLabel(value)
	case "", "default":
		return recommendVariantControl
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendResultLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendResultSuccess,
		recommendResultError,
		recommendResultEmpty,
		recommendResultFallback:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendStageLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendStageTotal,
		recommendStageSnapshotLookup,
		recommendStageRecall,
		recommendStageFeatureLoad,
		recommendStageCoarseRank,
		recommendStageFineRank,
		recommendStageRerank,
		recommendStageBuildItems,
		recommendStageSnapshotSave:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendRecallSourceLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendRecallSourceHot,
		recommendRecallSourceNewContent,
		recommendRecallSourceInterest:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendFallbackReasonLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendFallbackReasonDisabled,
		recommendFallbackReasonSnapshotMiss,
		recommendFallbackReasonSnapshotError,
		recommendFallbackReasonEmptyRecall,
		recommendFallbackReasonEnhancementError,
		recommendFallbackReasonHotError,
		recommendFallbackReasonBuildError,
		recommendFallbackReasonColdStart:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendSnapshotKindLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendSnapshotKindHot,
		recommendSnapshotKindPersonalized:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendSnapshotResultLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendSnapshotResultHit,
		recommendSnapshotResultMiss,
		recommendSnapshotResultError,
		recommendSnapshotResultSaved,
		recommendSnapshotResultSkipped:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendRerankRuleLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendRerankRuleAuthorWindow,
		recommendRerankRuleTypeWindow:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendErrorStageLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendErrorStageCandidateCache,
		recommendErrorStageRecall,
		recommendErrorStageFeatureLoad,
		recommendErrorStageSeenLoad,
		recommendErrorStageSnapshotSave,
		recommendErrorStageSnapshotRead,
		recommendErrorStageBuildItems,
		recommendErrorStageSeenWrite,
		recommendErrorStageProfileUpdate:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendProfileResultLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendProfileResultDisabled,
		recommendProfileResultSkipped,
		recommendProfileResultMiss,
		recommendProfileResultHit,
		recommendProfileResultError:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendTrackEventTypeLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case "click",
		"dwell",
		"exposure",
		"favorite",
		"follow",
		"like",
		"comment":
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func normalizeRecommendTrackEmitResultLabel(value string) string {
	switch canonicalRecommendMetricLabel(value) {
	case recommendResultSuccess,
		recommendResultError:
		return canonicalRecommendMetricLabel(value)
	default:
		return recommendMetricUnknownLabel
	}
}

func canonicalRecommendMetricLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}
