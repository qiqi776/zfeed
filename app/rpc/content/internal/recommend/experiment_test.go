package recommend

import (
	"testing"

	contentconfig "zfeed/app/rpc/content/internal/config"
)

func TestResolveExperimentVariantUsesDefaultWhenDisabled(t *testing.T) {
	exp := ExperimentConfig{
		ID:             "exp_rec_v1",
		Enabled:        false,
		Salt:           "salt",
		DefaultVariant: "control",
		Variants: []ExperimentVariant{
			{ID: "treatment", TrafficPermyriad: 10000},
		},
	}

	got := ResolveExperimentVariant(1001, exp)
	if got.ID != "control" {
		t.Fatalf("variant = %q, want control", got.ID)
	}
}

func TestResolveExperimentVariantSupportsPercentTraffic(t *testing.T) {
	exp := ExperimentConfig{
		ID:             "exp_rec_v1",
		Enabled:        true,
		Salt:           "salt",
		DefaultVariant: "control",
		Variants: []ExperimentVariant{
			{ID: "percent", TrafficPercent: 100},
		},
	}

	got := ResolveExperimentVariant(1001, exp)
	if got.ID != "percent" {
		t.Fatalf("variant = %q, want percent", got.ID)
	}
}

func TestResolveExperimentVariantSupportsPermyriadTraffic(t *testing.T) {
	exp := ExperimentConfig{
		ID:             "exp_rec_v1",
		Enabled:        true,
		Salt:           "salt",
		DefaultVariant: "control",
		Variants: []ExperimentVariant{
			{ID: "permyriad", TrafficPermyriad: 10000},
		},
	}

	got := ResolveExperimentVariant(1001, exp)
	if got.ID != "permyriad" {
		t.Fatalf("variant = %q, want permyriad", got.ID)
	}
}

func TestResolveExperimentVariantIsStableForUserAndSalt(t *testing.T) {
	exp := ExperimentConfig{
		ID:             "exp_rec_v1",
		Enabled:        true,
		Salt:           "salt",
		DefaultVariant: "control",
		Variants: []ExperimentVariant{
			{ID: "a", TrafficPermyriad: 5000},
			{ID: "b", TrafficPermyriad: 5000},
		},
	}

	first := ResolveExperimentVariant(1001, exp)
	for range 20 {
		got := ResolveExperimentVariant(1001, exp)
		if got.ID != first.ID {
			t.Fatalf("variant changed from %q to %q", first.ID, got.ID)
		}
	}
}

func TestApplyExperimentVariantOverridesRecommendConfig(t *testing.T) {
	cfg := NormalizeConfig(contentconfig.RecommendConfig{
		Hot: contentconfig.RecommendHotConfig{
			Weight: 0.55,
		},
		Interest: contentconfig.RecommendInterestConfig{
			Weight: 0.25,
		},
		Rank: contentconfig.RecommendRankConfig{
			BetaInterest: 0.30,
		},
		Diversity: contentconfig.RecommendDiversityConfig{
			AuthorWindow:       5,
			NewContentMinCount: 2,
		},
	})
	variant := ExperimentVariant{
		ID: "b",
		Overrides: map[string]string{
			"recall.hot.weight":               "0.40",
			"recall.interest.weight":          "0.35",
			"rank.beta_interest":              "0.40",
			"diversity.author_window":         "7",
			"diversity.new_content_min_count": "3",
		},
	}

	got := ApplyExperimentVariantOverrides(cfg, variant)

	if got.Hot.Weight != 0.40 {
		t.Fatalf("Hot.Weight = %f, want 0.40", got.Hot.Weight)
	}
	if got.Interest.Weight != 0.35 {
		t.Fatalf("Interest.Weight = %f, want 0.35", got.Interest.Weight)
	}
	if got.Rank.BetaInterest != 0.40 {
		t.Fatalf("Rank.BetaInterest = %f, want 0.40", got.Rank.BetaInterest)
	}
	if got.Diversity.AuthorWindow != 7 {
		t.Fatalf("Diversity.AuthorWindow = %d, want 7", got.Diversity.AuthorWindow)
	}
	if got.Diversity.NewContentMinCount != 3 {
		t.Fatalf("Diversity.NewContentMinCount = %d, want 3", got.Diversity.NewContentMinCount)
	}
}

func TestConfigHashChangesWithExperimentOverrides(t *testing.T) {
	base := NormalizeConfig(contentconfig.RecommendConfig{})
	changed := ApplyExperimentVariantOverrides(base, ExperimentVariant{
		ID: "b",
		Overrides: map[string]string{
			"rank.beta_interest": "0.40",
		},
	})

	baseHash := ConfigHash(base)
	changedHash := ConfigHash(changed)
	if baseHash == "" {
		t.Fatal("base hash is empty")
	}
	if changedHash == "" {
		t.Fatal("changed hash is empty")
	}
	if baseHash == changedHash {
		t.Fatalf("hash did not change after override: %s", baseHash)
	}
}
