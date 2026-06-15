package consumer

import (
	"strings"

	"github.com/zeromicro/go-zero/core/metric"

	"zfeed/app/rpc/content/internal/recommend/track"
)

const (
	recommendUserActionConsumeEventUnknown = "unknown"

	recommendUserActionConsumeResultSuccess        = "success"
	recommendUserActionConsumeResultParseError     = "parse_error"
	recommendUserActionConsumeResultProfileError   = "profile_error"
	recommendUserActionConsumeResultAggregateError = "aggregate_error"

	recommendTrackConsumeUnknownLabel = "unknown"

	recommendTrackConsumeVariantControl = "control"

	recommendTrackConsumeSourceRecommend   = "recommend"
	recommendTrackConsumeSourceInteraction = "interaction"
	recommendTrackConsumeSourceNewContent  = "new_content"
	recommendTrackConsumeSourceHot         = "hot"
	recommendTrackConsumeSourceInterest    = "interest"

	recommendTrackConsumeResultSuccess        = "success"
	recommendTrackConsumeResultParseError     = "parse_error"
	recommendTrackConsumeResultProfileError   = "profile_error"
	recommendTrackConsumeResultAggregateError = "aggregate_error"
)

var (
	recommendUserActionConsumeMetricLabels    = []string{"event_type", "result"}
	recommendTrackConsumeMetricLabels         = []string{"event_type", "variant", "source", "result"}
	recommendUserActionConsumeLagMetricLabels = []string{"event_type"}
	recommendTrackConsumeLagMetricLabels      = []string{"event_type", "source"}

	// PromQL: sum(rate(zfeed_recommend_user_action_consume_total[5m])) by (event_type, result)
	metricRecommendUserActionConsumeTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_user_action_consume",
		Name:      "total",
		Help:      "Recommendation user-action event consumption count.",
		Labels:    recommendUserActionConsumeMetricLabels,
	})

	// PromQL: sum(rate(zfeed_recommend_track_consume_total[5m])) by (event_type, variant, source, result)
	// PromQL: sum(rate(zfeed_recommend_track_consume_total{event_type="click",result="success"}[5m])) by (variant)
	//   / clamp_min(sum(rate(zfeed_recommend_track_consume_total{event_type="exposure",result="success"}[5m])) by (variant), 0.000001)
	// PromQL: 1000 * sum(rate(zfeed_recommend_track_consume_total{event_type=~"like|favorite|comment",result="success"}[5m])) by (variant)
	//   / clamp_min(sum(rate(zfeed_recommend_track_consume_total{event_type="exposure",result="success"}[5m])) by (variant), 0.000001)
	metricRecommendTrackConsumeTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_track_consume",
		Name:      "total",
		Help:      "Recommendation track event consumption count.",
		Labels:    recommendTrackConsumeMetricLabels,
	})

	// PromQL: histogram_quantile(0.95, sum(rate(zfeed_recommend_user_action_consume_lag_seconds_bucket[5m])) by (le, event_type))
	metricRecommendUserActionConsumeLagSeconds = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_user_action_consume",
		Name:      "lag_seconds",
		Help:      "Recommendation user-action event consume lag in seconds.",
		Labels:    recommendUserActionConsumeLagMetricLabels,
		Buckets:   []float64{0, 0.1, 0.5, 1, 3, 5, 10, 30, 60, 300, 900},
	})

	// PromQL: histogram_quantile(0.95, sum(rate(zfeed_recommend_track_consume_lag_seconds_bucket[5m])) by (le, event_type, source))
	metricRecommendTrackConsumeLagSeconds = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_track_consume",
		Name:      "lag_seconds",
		Help:      "Recommendation track event consume lag in seconds.",
		Labels:    recommendTrackConsumeLagMetricLabels,
		Buckets:   []float64{0, 0.1, 0.5, 1, 3, 5, 10, 30, 60, 300, 900},
	})

	recordRecommendUserActionConsumeMetric     = recordRecommendUserActionConsume
	recordRecommendTrackConsumeMetric          = recordRecommendTrackConsume
	observeRecommendUserActionConsumeLagMetric = observeRecommendUserActionConsumeLag
	observeRecommendTrackConsumeLagMetric      = observeRecommendTrackConsumeLag
)

func recordRecommendUserActionConsume(eventType, result string) {
	if metricRecommendUserActionConsumeTotal == nil {
		return
	}

	metricRecommendUserActionConsumeTotal.Inc(
		normalizeRecommendUserActionConsumeEventTypeLabel(eventType),
		normalizeRecommendUserActionConsumeResultLabel(result),
	)
}

func recordRecommendTrackConsume(eventType, variant, source, result string) {
	if metricRecommendTrackConsumeTotal == nil {
		return
	}

	metricRecommendTrackConsumeTotal.Inc(
		normalizeRecommendTrackConsumeEventTypeLabel(eventType),
		normalizeRecommendTrackConsumeVariantLabel(variant),
		normalizeRecommendTrackConsumeSourceLabel(source),
		normalizeRecommendTrackConsumeResultLabel(result),
	)
}

func observeRecommendUserActionConsumeLag(eventType string, seconds float64) {
	if metricRecommendUserActionConsumeLagSeconds == nil {
		return
	}

	metricRecommendUserActionConsumeLagSeconds.ObserveFloat(
		normalizeRecommendConsumeLagSeconds(seconds),
		normalizeRecommendUserActionConsumeEventTypeLabel(eventType),
	)
}

func observeRecommendTrackConsumeLag(eventType, source string, seconds float64) {
	if metricRecommendTrackConsumeLagSeconds == nil {
		return
	}

	metricRecommendTrackConsumeLagSeconds.ObserveFloat(
		normalizeRecommendConsumeLagSeconds(seconds),
		normalizeRecommendTrackConsumeEventTypeLabel(eventType),
		normalizeRecommendTrackConsumeSourceLabel(source),
	)
}

func normalizeRecommendConsumeLagSeconds(seconds float64) float64 {
	if seconds < 0 {
		return 0
	}
	return seconds
}

func normalizeRecommendUserActionConsumeEventTypeLabel(value string) string {
	switch canonicalRecommendUserActionMetricLabel(value) {
	case track.EventTypeLike,
		track.EventTypeUnlike,
		track.EventTypeFavorite,
		track.EventTypeUnfavorite,
		track.EventTypeComment:
		return canonicalRecommendUserActionMetricLabel(value)
	default:
		return recommendUserActionConsumeEventUnknown
	}
}

func normalizeRecommendUserActionConsumeResultLabel(value string) string {
	switch canonicalRecommendUserActionMetricLabel(value) {
	case recommendUserActionConsumeResultSuccess,
		recommendUserActionConsumeResultParseError,
		recommendUserActionConsumeResultProfileError,
		recommendUserActionConsumeResultAggregateError:
		return canonicalRecommendUserActionMetricLabel(value)
	default:
		return recommendUserActionConsumeEventUnknown
	}
}

func canonicalRecommendUserActionMetricLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func normalizeRecommendTrackConsumeEventTypeLabel(value string) string {
	switch canonicalRecommendTrackConsumeMetricLabel(value) {
	case track.EventTypeExposure,
		track.EventTypeClick,
		track.EventTypeDwell,
		track.EventTypeLike,
		track.EventTypeFavorite,
		track.EventTypeComment,
		track.EventTypeUnlike,
		track.EventTypeUnfavorite:
		return canonicalRecommendTrackConsumeMetricLabel(value)
	default:
		return recommendTrackConsumeUnknownLabel
	}
}

func normalizeRecommendTrackConsumeVariantLabel(value string) string {
	switch canonicalRecommendTrackConsumeMetricLabel(value) {
	case "a", "b", recommendTrackConsumeVariantControl:
		return canonicalRecommendTrackConsumeMetricLabel(value)
	case "", "default":
		return recommendTrackConsumeVariantControl
	default:
		return recommendTrackConsumeUnknownLabel
	}
}

func normalizeRecommendTrackConsumeSourceLabel(value string) string {
	switch canonicalRecommendTrackConsumeMetricLabel(value) {
	case recommendTrackConsumeSourceRecommend,
		recommendTrackConsumeSourceInteraction,
		recommendTrackConsumeSourceNewContent,
		recommendTrackConsumeSourceHot,
		recommendTrackConsumeSourceInterest:
		return canonicalRecommendTrackConsumeMetricLabel(value)
	default:
		return recommendTrackConsumeUnknownLabel
	}
}

func normalizeRecommendTrackConsumeResultLabel(value string) string {
	switch canonicalRecommendTrackConsumeMetricLabel(value) {
	case recommendTrackConsumeResultSuccess,
		recommendTrackConsumeResultParseError,
		recommendTrackConsumeResultProfileError,
		recommendTrackConsumeResultAggregateError:
		return canonicalRecommendTrackConsumeMetricLabel(value)
	default:
		return recommendTrackConsumeUnknownLabel
	}
}

func canonicalRecommendTrackConsumeMetricLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}
