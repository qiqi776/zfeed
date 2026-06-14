package recommend

import "testing"

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
