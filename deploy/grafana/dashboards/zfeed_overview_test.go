package dashboards

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type overviewDashboard struct {
	Panels []overviewPanel `json:"panels"`
}

type overviewPanel struct {
	Title   string           `json:"title"`
	Type    string           `json:"type"`
	Targets []overviewTarget `json:"targets"`
}

type overviewTarget struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
}

func TestZFeedOverviewIncludesUserActionMigrationPanels(t *testing.T) {
	dashboard := loadOverviewDashboard(t)

	assertPanelTarget(t, dashboard, "User Action Outbox Dispatch",
		"sum by (action, result) (rate(zfeed_user_action_outbox_total[5m]))",
		"{{action}} / {{result}}",
	)
	assertPanelTarget(t, dashboard, "Recommendation User Action Consume",
		"sum by (event_type, result) (rate(zfeed_recommend_user_action_consume_total[5m]))",
		"{{event_type}} / {{result}}",
	)
}

func loadOverviewDashboard(t *testing.T) overviewDashboard {
	t.Helper()

	data, err := os.ReadFile("zfeed-overview.json")
	if err != nil {
		t.Fatalf("read dashboard: %v", err)
	}

	var dashboard overviewDashboard
	if err := json.Unmarshal(data, &dashboard); err != nil {
		t.Fatalf("parse dashboard json: %v", err)
	}
	if len(dashboard.Panels) == 0 {
		t.Fatal("dashboard has no panels")
	}

	return dashboard
}

func assertPanelTarget(t *testing.T, dashboard overviewDashboard, title, expr, legendFormat string) {
	t.Helper()

	for _, panel := range dashboard.Panels {
		if panel.Title != title {
			continue
		}
		if panel.Type != "timeseries" {
			t.Fatalf("panel %q type = %q, want timeseries", title, panel.Type)
		}
		if len(panel.Targets) != 1 {
			t.Fatalf("panel %q has %d targets, want 1", title, len(panel.Targets))
		}

		target := panel.Targets[0]
		if target.Expr != expr {
			t.Fatalf("panel %q expr = %q, want %q", title, target.Expr, expr)
		}
		if target.LegendFormat != legendFormat {
			t.Fatalf("panel %q legend = %q, want %q", title, target.LegendFormat, legendFormat)
		}
		assertLowCardinalityPromQL(t, title, target.Expr)
		return
	}

	t.Fatalf("missing panel %q", title)
}

func assertLowCardinalityPromQL(t *testing.T, title, expr string) {
	t.Helper()

	for _, label := range []string{"user_id", "content_id", "target_id"} {
		if strings.Contains(expr, label) {
			t.Fatalf("panel %q expr uses high-cardinality label %q: %s", title, label, expr)
		}
	}
}
