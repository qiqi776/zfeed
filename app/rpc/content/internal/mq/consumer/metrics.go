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
	recommendUserActionConsumeMetricLabels = []string{"event_type", "result"}
	recommendTrackConsumeMetricLabels      = []string{"event_type", "variant", "source", "result"}

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

	recordRecommendUserActionConsumeMetric = recordRecommendUserActionConsume
	recordRecommendTrackConsumeMetric      = recordRecommendTrackConsume
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
