package feedlogic

import "testing"

func TestRecommendMetricLabelsExcludeHighCardinalityIDs(t *testing.T) {
	metricLabels := map[string][]string{
		"zfeed_recommend_requests_total":         recommendRequestMetricLabels,
		"zfeed_recommend_stage_duration_seconds": recommendStageDurationMetricLabels,
		"zfeed_recommend_recall_items_total":     recommendRecallItemsMetricLabels,
		"zfeed_recommend_fallback_total":         recommendFallbackMetricLabels,
		"zfeed_recommend_snapshot_total":         recommendSnapshotMetricLabels,
		"zfeed_recommend_track_emit_total":       recommendTrackMetricLabels,
	}

	for metricName, labels := range metricLabels {
		t.Run(metricName, func(t *testing.T) {
			for _, label := range labels {
				switch label {
				case "user_id", "content_id", "snapshot_id":
					t.Fatalf("metric label %q must not include high-cardinality label %q", metricName, label)
				}
			}
		})
	}
}

func TestRecommendMetricLabelNormalizersClampUnknownValues(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		in   string
		want string
	}{
		{name: "mode allowed", fn: normalizeRecommendModeLabel, in: " Personalized ", want: "personalized"},
		{name: "mode unknown", fn: normalizeRecommendModeLabel, in: "user_123456", want: recommendMetricUnknownLabel},
		{name: "variant allowed", fn: normalizeRecommendVariantLabel, in: " A ", want: "a"},
		{name: "variant unknown", fn: normalizeRecommendVariantLabel, in: "exp-user-123456", want: recommendMetricUnknownLabel},
		{name: "result allowed", fn: normalizeRecommendResultLabel, in: " empty ", want: "empty"},
		{name: "result unknown", fn: normalizeRecommendResultLabel, in: "content_987654", want: recommendMetricUnknownLabel},
		{name: "stage allowed", fn: normalizeRecommendStageLabel, in: " feature_load ", want: "feature_load"},
		{name: "stage unknown", fn: normalizeRecommendStageLabel, in: "snapshot-id-123", want: recommendMetricUnknownLabel},
		{name: "source allowed", fn: normalizeRecommendRecallSourceLabel, in: " new_content ", want: "new_content"},
		{name: "source unknown", fn: normalizeRecommendRecallSourceLabel, in: "content-123", want: recommendMetricUnknownLabel},
		{name: "fallback reason allowed", fn: normalizeRecommendFallbackReasonLabel, in: " snapshot_miss ", want: "snapshot_miss"},
		{name: "fallback reason unknown", fn: normalizeRecommendFallbackReasonLabel, in: "user_123", want: recommendMetricUnknownLabel},
		{name: "snapshot kind allowed", fn: normalizeRecommendSnapshotKindLabel, in: " Hot ", want: "hot"},
		{name: "snapshot kind unknown", fn: normalizeRecommendSnapshotKindLabel, in: "snap-123", want: recommendMetricUnknownLabel},
		{name: "snapshot result allowed", fn: normalizeRecommendSnapshotResultLabel, in: " hit ", want: "hit"},
		{name: "snapshot result unknown", fn: normalizeRecommendSnapshotResultLabel, in: "snap-123", want: recommendMetricUnknownLabel},
		{name: "track event type allowed", fn: normalizeRecommendTrackEventTypeLabel, in: " Exposure ", want: "exposure"},
		{name: "track event type unknown", fn: normalizeRecommendTrackEventTypeLabel, in: "user_123", want: recommendMetricUnknownLabel},
		{name: "track emit result allowed", fn: normalizeRecommendTrackEmitResultLabel, in: " Success ", want: "success"},
		{name: "track emit result unknown", fn: normalizeRecommendTrackEmitResultLabel, in: "snapshot-id-123", want: recommendMetricUnknownLabel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.in); got != tt.want {
				t.Fatalf("normalizer(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRecommendMetricRecordHelpersAcceptUnknownValues(t *testing.T) {
	recordRecommendRequest("user_123456", "exp-user-123456", "content_987654")
	recordRecommendStageDuration("snapshot-id-123", "exp-user-123456", -1)
	recordRecommendRecallItems("content-123", "exp-user-123456", -1)
	recordRecommendFallback("user_123")
	recordRecommendSnapshot("snap-123", "snap-123")
	recordRecommendTrackEmit("user_123", "snap-123")
}
