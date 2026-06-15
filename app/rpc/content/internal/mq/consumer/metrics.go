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
)

var (
	recommendUserActionConsumeMetricLabels = []string{"event_type", "result"}

	// PromQL: sum(rate(zfeed_recommend_user_action_consume_total[5m])) by (event_type, result)
	metricRecommendUserActionConsumeTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "recommend_user_action_consume",
		Name:      "total",
		Help:      "Recommendation user-action event consumption count.",
		Labels:    recommendUserActionConsumeMetricLabels,
	})

	recordRecommendUserActionConsumeMetric = recordRecommendUserActionConsume
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
