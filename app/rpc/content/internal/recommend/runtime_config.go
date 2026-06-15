package recommend

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentconfig "zfeed/app/rpc/content/internal/config"
)

const RuntimeFlagKey = "rec:flag:recommend"

const runtimeConfigCacheTTL = 10 * time.Second

var (
	runtimeConfigNow = time.Now

	runtimeConfigCacheMu sync.RWMutex
	runtimeConfigCache   = map[string]runtimeConfigCacheEntry{}
)

type runtimeConfigCacheEntry struct {
	cfg       contentconfig.RecommendConfig
	expiresAt time.Time
}

func LoadRuntimeConfig(
	ctx context.Context,
	rds *gzredis.Redis,
	base contentconfig.RecommendConfig,
) (contentconfig.RecommendConfig, error) {
	cfg := NormalizeConfig(base)
	if rds == nil {
		return cfg, nil
	}

	cacheKey := buildRuntimeConfigCacheKey(rds, cfg)
	if cached, ok := getRuntimeConfigCache(cacheKey); ok {
		return cached, nil
	}

	raw, err := rds.HgetallCtx(ctx, RuntimeFlagKey)
	if err != nil {
		return cfg, err
	}
	ApplyRuntimeOverrides(&cfg, raw)
	setRuntimeConfigCache(cacheKey, cfg)
	return cfg, nil
}

func getRuntimeConfigCache(cacheKey string) (contentconfig.RecommendConfig, bool) {
	now := runtimeConfigNow()
	runtimeConfigCacheMu.RLock()
	entry, ok := runtimeConfigCache[cacheKey]
	runtimeConfigCacheMu.RUnlock()
	if !ok || !now.Before(entry.expiresAt) {
		return contentconfig.RecommendConfig{}, false
	}
	return entry.cfg, true
}

func setRuntimeConfigCache(cacheKey string, cfg contentconfig.RecommendConfig) {
	runtimeConfigCacheMu.Lock()
	defer runtimeConfigCacheMu.Unlock()

	runtimeConfigCache[cacheKey] = runtimeConfigCacheEntry{
		cfg:       cfg,
		expiresAt: runtimeConfigNow().Add(runtimeConfigCacheTTL),
	}
}

func buildRuntimeConfigCacheKey(rds *gzredis.Redis, cfg contentconfig.RecommendConfig) string {
	return fmt.Sprintf("%p:%#v", rds, cfg)
}

func resetRuntimeConfigCacheForTest() {
	runtimeConfigCacheMu.Lock()
	defer runtimeConfigCacheMu.Unlock()

	runtimeConfigCache = map[string]runtimeConfigCacheEntry{}
	runtimeConfigNow = time.Now
}

func ApplyRuntimeOverrides(cfg *contentconfig.RecommendConfig, raw map[string]string) {
	if cfg == nil || len(raw) == 0 {
		return
	}

	if value, ok := parseBool(raw["enabled"]); ok {
		cfg.Enabled = value
	}
	if value, ok := parseBool(raw["fallback_to_hot"]); ok {
		cfg.FallbackToHot = value
	}
	if value, ok := parseBool(raw["recall.hot.enabled"]); ok {
		cfg.Hot.Enabled = value
	}
	if value, ok := parseNonNegativeFloat(raw["recall.hot.weight"]); ok {
		cfg.Hot.Weight = value
	}
	if value, ok := parsePositiveInt(raw["recall.hot.limit"]); ok {
		cfg.Hot.Limit = value
	}
	if value, ok := parseBool(raw["recall.new_content.enabled"]); ok {
		cfg.NewContent.Enabled = value
	}
	if value, ok := parseNonNegativeFloat(raw["recall.new_content.weight"]); ok {
		cfg.NewContent.Weight = value
	}
	if value, ok := parsePositiveInt(raw["recall.new_content.limit"]); ok {
		cfg.NewContent.Limit = value
	}
	if value, ok := parseBool(raw["recall.interest.enabled"]); ok {
		cfg.Interest.Enabled = value
	}
	if value, ok := parseNonNegativeFloat(raw["recall.interest.weight"]); ok {
		cfg.Interest.Weight = value
	}
	if value, ok := parsePositiveInt(raw["recall.interest.limit"]); ok {
		cfg.Interest.Limit = value
	}
	if value, ok := parsePositiveInt(raw["rank.coarse_limit"]); ok {
		cfg.Rank.CoarseLimit = value
	}
	if value, ok := parseNonNegativeFloat(raw["rank.alpha_hot"]); ok {
		cfg.Rank.AlphaHot = value
	}
	if value, ok := parseNonNegativeFloat(raw["rank.beta_interest"]); ok {
		cfg.Rank.BetaInterest = value
	}
	if value, ok := parseNonNegativeFloat(raw["rank.gamma_fresh"]); ok {
		cfg.Rank.GammaFresh = value
	}
	if value, ok := parseNonNegativeFloat(raw["rank.delta_quality"]); ok {
		cfg.Rank.DeltaQuality = value
	}
	if value, ok := parseNonNegativeFloat(raw["rank.seen_penalty"]); ok {
		cfg.Rank.SeenPenalty = value
	}
	if value, ok := parseBool(raw["diversity.enabled"]); ok {
		cfg.Diversity.Enabled = value
	}
	if value, ok := parsePositiveInt(raw["diversity.author_window"]); ok {
		cfg.Diversity.AuthorWindow = value
	}
	if value, ok := parsePositiveInt(raw["diversity.max_same_author"]); ok {
		cfg.Diversity.MaxSameAuthor = value
	}
	if value, ok := parsePositiveInt(raw["diversity.type_window"]); ok {
		cfg.Diversity.TypeWindow = value
	}
	if value, ok := parsePositiveInt(raw["diversity.max_same_type"]); ok {
		cfg.Diversity.MaxSameType = value
	}
	if value, ok := parsePositiveInt(raw["diversity.new_content_top_n"]); ok {
		cfg.Diversity.NewContentTopN = value
	}
	if value, ok := parsePositiveInt(raw["diversity.new_content_min_count"]); ok {
		cfg.Diversity.NewContentMinCount = value
	}
}

func parseBool(raw string) (bool, bool) {
	if strings.TrimSpace(raw) == "" {
		return false, false
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err == nil {
		return value, true
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "yes", "y", "on":
		return true, true
	case "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func parseNonNegativeFloat(raw string) (float64, bool) {
	if strings.TrimSpace(raw) == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func parsePositiveInt(raw string) (int, bool) {
	if strings.TrimSpace(raw) == "" {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}
