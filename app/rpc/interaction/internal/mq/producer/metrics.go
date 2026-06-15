package producer

import (
	"strings"

	"github.com/zeromicro/go-zero/core/metric"

	"zfeed/app/rpc/interaction/internal/mq/event"
)

const (
	userActionOutboxLabelUnknown = "unknown"

	userActionOutboxResultSent       = "sent"
	userActionOutboxResultRetry      = "retry"
	userActionOutboxResultReplayed   = "replayed"
	userActionOutboxResultMarkFailed = "mark_failed"
)

var (
	userActionOutboxMetricLabels = []string{"action", "result"}

	// PromQL: sum(rate(zfeed_user_action_outbox_total[5m])) by (action, result)
	metricUserActionOutboxTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "zfeed",
		Subsystem: "user_action_outbox",
		Name:      "total",
		Help:      "Interaction user-action outbox dispatch count.",
		Labels:    userActionOutboxMetricLabels,
	})

	recordUserActionOutboxMetric = recordUserActionOutbox
)

func recordUserActionOutbox(action, result string) {
	if metricUserActionOutboxTotal == nil {
		return
	}

	metricUserActionOutboxTotal.Inc(
		normalizeUserActionOutboxActionLabel(action),
		normalizeUserActionOutboxResultLabel(result),
	)
}

func normalizeUserActionOutboxActionLabel(value string) string {
	switch canonicalUserActionOutboxMetricLabel(value) {
	case event.UserActionLike,
		event.UserActionUnlike,
		event.UserActionFavorite,
		event.UserActionUnfavorite,
		event.UserActionComment:
		return canonicalUserActionOutboxMetricLabel(value)
	default:
		return userActionOutboxLabelUnknown
	}
}

func normalizeUserActionOutboxResultLabel(value string) string {
	switch canonicalUserActionOutboxMetricLabel(value) {
	case userActionOutboxResultSent,
		userActionOutboxResultRetry,
		userActionOutboxResultReplayed,
		userActionOutboxResultMarkFailed:
		return canonicalUserActionOutboxMetricLabel(value)
	default:
		return userActionOutboxLabelUnknown
	}
}

func canonicalUserActionOutboxMetricLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}
