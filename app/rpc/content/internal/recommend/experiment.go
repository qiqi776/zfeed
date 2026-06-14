package recommend

import (
	"fmt"

	"github.com/spaolacci/murmur3"
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
