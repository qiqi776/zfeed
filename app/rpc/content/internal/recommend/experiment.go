package recommend

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/spaolacci/murmur3"

	contentconfig "zfeed/app/rpc/content/internal/config"
)

type ExperimentConfig struct {
	ID               string
	Enabled          bool
	Salt             string
	TrafficPermyriad int
	TrafficPercent   int
	DefaultVariant   string
	Variants         []ExperimentVariant
}

type ExperimentVariant struct {
	ID               string
	TrafficPermyriad int
	TrafficPercent   int
	Overrides        map[string]string
}

func ExperimentConfigFromContent(cfg contentconfig.RecommendExperimentConfig) ExperimentConfig {
	variants := make([]ExperimentVariant, 0, len(cfg.Variants))
	for _, variant := range cfg.Variants {
		overrides := make(map[string]string, len(variant.Overrides))
		for key, value := range variant.Overrides {
			overrides[key] = value
		}
		variants = append(variants, ExperimentVariant{
			ID:               variant.ID,
			TrafficPermyriad: variant.TrafficPermyriad,
			TrafficPercent:   variant.TrafficPercent,
			Overrides:        overrides,
		})
	}

	return ExperimentConfig{
		ID:               cfg.ID,
		Enabled:          cfg.Enabled,
		Salt:             cfg.Salt,
		TrafficPermyriad: cfg.TrafficPermyriad,
		TrafficPercent:   cfg.TrafficPercent,
		DefaultVariant:   cfg.DefaultVariant,
		Variants:         variants,
	}
}

func ResolveExperimentVariant(userID int64, exp ExperimentConfig) ExperimentVariant {
	if userID <= 0 || !exp.Enabled {
		return defaultVariant(exp)
	}

	bucket := experimentBucket(exp, userID)
	experimentTraffic := trafficPermyriad(exp.TrafficPermyriad, exp.TrafficPercent)
	if experimentTraffic == 0 {
		experimentTraffic = 10000
	}
	if bucket >= experimentTraffic {
		return defaultVariant(exp)
	}

	var cursor int
	for _, variant := range exp.Variants {
		cursor += trafficPermyriad(variant.TrafficPermyriad, variant.TrafficPercent)
		if cursor <= 0 {
			continue
		}
		if bucket < cursor {
			return variant
		}
	}
	return defaultVariant(exp)
}

func experimentBucket(exp ExperimentConfig, userID int64) int {
	input := fmt.Sprintf("%s:%d:%s", exp.ID, userID, exp.Salt)
	return int(murmur3.Sum32([]byte(input)) % 10000)
}

func defaultVariant(exp ExperimentConfig) ExperimentVariant {
	for _, variant := range exp.Variants {
		if variant.ID == exp.DefaultVariant {
			return variant
		}
	}
	return ExperimentVariant{ID: exp.DefaultVariant}
}

func trafficPermyriad(permyriad, percent int) int {
	if permyriad <= 0 && percent > 0 {
		permyriad = percent * 100
	}
	if permyriad < 0 {
		return 0
	}
	if permyriad > 10000 {
		return 10000
	}
	return permyriad
}

func ApplyExperimentVariantOverrides(
	cfg contentconfig.RecommendConfig,
	variant ExperimentVariant,
) contentconfig.RecommendConfig {
	if len(variant.Overrides) == 0 {
		return cfg
	}
	ApplyRuntimeOverrides(&cfg, variant.Overrides)
	return NormalizeConfig(cfg)
}

func ConfigHash(cfg contentconfig.RecommendConfig) string {
	normalized := NormalizeConfig(cfg)
	sum := sha1.Sum([]byte(fmt.Sprintf("%#v", normalized)))
	return hex.EncodeToString(sum[:])[:8]
}
