package rules

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type alertRulesFile struct {
	Groups []alertRuleGroup `yaml:"groups"`
}

type alertRuleGroup struct {
	Name  string      `yaml:"name"`
	Rules []alertRule `yaml:"rules"`
}

type alertRule struct {
	Alert       string            `yaml:"alert"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

func TestZfeedRecommendAlertsCoverMigrationRisks(t *testing.T) {
	rulesFile := loadRecommendAlertRules(t)

	tests := []struct {
		name      string
		exprPart  string
		severity  string
		forWindow string
	}{
		{
			name: "ZfeedRecommendTrackConsumeLagHigh",
			exprPart: "histogram_quantile(0.95, sum by (le, event_type, source) " +
				"(rate(zfeed_recommend_track_consume_lag_seconds_bucket[5m]))) > 60",
			severity:  "warning",
			forWindow: "10m",
		},
		{
			name: "ZfeedRecommendUserActionConsumeLagHigh",
			exprPart: "histogram_quantile(0.95, sum by (le, event_type) " +
				"(rate(zfeed_recommend_user_action_consume_lag_seconds_bucket[5m]))) > 60",
			severity:  "warning",
			forWindow: "10m",
		},
		{
			name: "ZfeedRecommendTrackConsumeErrorRateHigh",
			exprPart: "sum(rate(zfeed_recommend_track_consume_total{result=~\"parse_error|profile_error|aggregate_error\"}[5m])) " +
				"/ clamp_min(sum(rate(zfeed_recommend_track_consume_total[5m])), 0.000001) > 0.01",
			severity:  "warning",
			forWindow: "10m",
		},
		{
			name:      "ZfeedUserActionOutboxFailureHigh",
			exprPart:  "sum(rate(zfeed_user_action_outbox_total{result=~\"retry|mark_failed\"}[5m])) > 0",
			severity:  "warning",
			forWindow: "10m",
		},
		{
			name: "ZfeedRecommendFallbackRateHigh",
			exprPart: "sum(rate(zfeed_recommend_fallback_total[5m])) " +
				"/ clamp_min(sum(rate(zfeed_recommend_requests_total[5m])), 0.000001) > 0.2",
			severity:  "warning",
			forWindow: "15m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := findAlertRule(t, rulesFile, tt.name)
			if !strings.Contains(oneLine(rule.Expr), tt.exprPart) {
				t.Fatalf("alert %s expr = %q, want to contain %q", tt.name, oneLine(rule.Expr), tt.exprPart)
			}
			if rule.For != tt.forWindow {
				t.Fatalf("alert %s for = %q, want %q", tt.name, rule.For, tt.forWindow)
			}
			if rule.Labels["severity"] != tt.severity {
				t.Fatalf("alert %s severity = %q, want %q", tt.name, rule.Labels["severity"], tt.severity)
			}
			if rule.Labels["scope"] != "recommend" {
				t.Fatalf("alert %s scope = %q, want recommend", tt.name, rule.Labels["scope"])
			}
			if strings.TrimSpace(rule.Annotations["summary"]) == "" ||
				strings.TrimSpace(rule.Annotations["description"]) == "" {
				t.Fatalf("alert %s must include summary and description annotations", tt.name)
			}
			assertLowCardinalityAlertExpr(t, tt.name, rule.Expr)
		})
	}
}

func loadRecommendAlertRules(t *testing.T) alertRulesFile {
	t.Helper()

	data, err := os.ReadFile("zfeed-recommend-alerts.yml")
	if err != nil {
		t.Fatalf("read recommend alert rules: %v", err)
	}

	var rulesFile alertRulesFile
	if err := yaml.Unmarshal(data, &rulesFile); err != nil {
		t.Fatalf("parse recommend alert rules: %v", err)
	}
	if len(rulesFile.Groups) != 1 {
		t.Fatalf("recommend alert groups = %+v, want one group", rulesFile.Groups)
	}
	if rulesFile.Groups[0].Name != "zfeed-recommend" {
		t.Fatalf("recommend alert group = %q, want zfeed-recommend", rulesFile.Groups[0].Name)
	}
	return rulesFile
}

func findAlertRule(t *testing.T, rulesFile alertRulesFile, name string) alertRule {
	t.Helper()

	for _, group := range rulesFile.Groups {
		for _, rule := range group.Rules {
			if rule.Alert == name {
				return rule
			}
		}
	}
	t.Fatalf("missing alert %q", name)
	return alertRule{}
}

func assertLowCardinalityAlertExpr(t *testing.T, alertName, expr string) {
	t.Helper()

	for _, label := range []string{"user_id", "content_id", "target_id", "event_id", "snapshot_id"} {
		if strings.Contains(expr, label) {
			t.Fatalf("alert %s expr uses high-cardinality label %q: %s", alertName, label, expr)
		}
	}
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
