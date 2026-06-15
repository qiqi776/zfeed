package recommend

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

const (
	ActionLike       = "like"
	ActionFavorite   = "favorite"
	ActionComment    = "comment"
	ActionClick      = "click"
	ActionDwell      = "dwell"
	ActionUnlike     = "unlike"
	ActionUnfavorite = "unfavorite"

	MinProfileDwellMs int64 = 10_000

	profileDecayWindowHours = 168
	maxProfileTags          = 50
)

type ProfileEvent struct {
	EventID   string
	EventType string
	UserID    int64
	ContentID int64
	Weight    float64
}

func ApplyProfileEvent(ctx context.Context, rds *redis.Redis, cfg contentconfig.RecommendConfig, event ProfileEvent) error {
	if rds == nil || event.UserID <= 0 || event.ContentID <= 0 || event.EventID == "" {
		return nil
	}
	cfg = NormalizeConfig(cfg)
	ok, err := rds.SetnxExCtx(ctx, "rec:profile:event:"+event.EventID, "1", 24*3600)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	tags, err := LoadContentTags(ctx, rds, event.ContentID)
	if err != nil || len(tags) == 0 {
		return err
	}

	delta := event.Weight
	if delta == 0 {
		delta = actionWeight(event.EventType)
	}
	if delta == 0 {
		return nil
	}

	profileKey := redisconsts.BuildRecommendUserProfileKey(event.UserID)
	now := time.Now()
	decayFactor, err := loadProfileDecayFactor(ctx, rds, profileKey, now)
	if err != nil {
		return err
	}
	for tag, weight := range tags {
		currentRaw, err := rds.HgetCtx(ctx, profileKey, tag)
		if err != nil && !isRedisNil(err) {
			return err
		}
		current, _ := strconv.ParseFloat(currentRaw, 64)
		next := current*decayFactor + delta*weight
		if err := rds.HsetCtx(ctx, profileKey, tag, strconv.FormatFloat(next, 'f', 6, 64)); err != nil {
			return err
		}
	}
	if err := rds.HsetCtx(ctx, profileKey, "_updated_at", strconv.FormatInt(now.Unix(), 10)); err != nil {
		return err
	}
	if err := trimProfileTags(ctx, rds, profileKey, maxProfileTags); err != nil {
		return err
	}
	if err := updateProfileMetadata(ctx, rds, profileKey, event.EventID, delta > 0, now); err != nil {
		return err
	}
	return rds.ExpireCtx(ctx, profileKey, cfg.Interest.ProfileTTL)
}

func LoadUserProfile(ctx context.Context, rds *redis.Redis, userID int64, cfg contentconfig.RecommendInterestConfig) (map[string]float64, error) {
	if rds == nil || userID <= 0 {
		return map[string]float64{}, nil
	}
	cfg = NormalizeConfig(contentconfig.RecommendConfig{Interest: cfg}).Interest

	raw, err := rds.HgetallCtx(ctx, redisconsts.BuildRecommendUserProfileKey(userID))
	if err != nil {
		return nil, err
	}
	if isProfileInactive(raw, cfg.ProfileTTL, time.Now()) {
		return map[string]float64{}, nil
	}
	tags := positiveWeights(parseWeights(raw))
	if len(tags) < cfg.MinTags {
		return map[string]float64{}, nil
	}
	return topWeights(tags, cfg.TopTags), nil
}

func positiveWeights(weights map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(weights))
	for tag, weight := range weights {
		if weight <= 0 {
			continue
		}
		result[tag] = weight
	}
	return result
}

func isProfileInactive(raw map[string]string, activeWindowSeconds int, now time.Time) bool {
	if activeWindowSeconds <= 0 {
		return false
	}
	lastActiveRaw := raw["_last_active_at"]
	if lastActiveRaw == "" {
		return false
	}

	lastActiveAt, err := strconv.ParseInt(lastActiveRaw, 10, 64)
	if err != nil || lastActiveAt <= 0 {
		return false
	}
	return now.Unix()-lastActiveAt > int64(activeWindowSeconds)
}

func ProfileActionForTrackEvent(eventType string, dwellMs int64) (string, bool) {
	switch eventType {
	case ActionClick,
		ActionLike,
		ActionFavorite,
		ActionComment,
		ActionUnlike,
		ActionUnfavorite:
		return eventType, true
	case ActionDwell:
		if dwellMs < MinProfileDwellMs {
			return "", false
		}
		return eventType, true
	default:
		return "", false
	}
}

func actionWeight(eventType string) float64 {
	switch eventType {
	case ActionLike:
		return 1
	case ActionFavorite:
		return 3
	case ActionComment:
		return 2
	case ActionClick:
		return 0.5
	case ActionDwell:
		return 0.8
	case ActionUnlike:
		return -0.8
	case ActionUnfavorite:
		return -1.5
	default:
		return 0
	}
}

func loadProfileDecayFactor(ctx context.Context, rds *redis.Redis, profileKey string, now time.Time) (float64, error) {
	updatedAtRaw, err := rds.HgetCtx(ctx, profileKey, "_updated_at")
	if err != nil {
		if isRedisNil(err) {
			return 1, nil
		}
		return 0, err
	}

	updatedAt, err := strconv.ParseInt(updatedAtRaw, 10, 64)
	if err != nil || updatedAt <= 0 {
		return 1, nil
	}

	elapsedHours := float64(now.Unix()-updatedAt) / 3600
	if elapsedHours <= 0 {
		return 1, nil
	}
	return math.Exp(-elapsedHours / profileDecayWindowHours), nil
}

func trimProfileTags(ctx context.Context, rds *redis.Redis, profileKey string, limit int) error {
	if rds == nil || profileKey == "" || limit <= 0 {
		return nil
	}

	raw, err := rds.HgetallCtx(ctx, profileKey)
	if err != nil {
		return err
	}
	weights := parseWeights(raw)
	if len(weights) <= limit {
		return nil
	}

	kept := topWeights(weights, limit)
	fields := make([]string, 0, len(weights)-len(kept))
	for tag := range weights {
		if _, ok := kept[tag]; ok {
			continue
		}
		fields = append(fields, tag)
	}
	if len(fields) == 0 {
		return nil
	}

	_, err = rds.HdelCtx(ctx, profileKey, fields...)
	return err
}

func updateProfileMetadata(
	ctx context.Context,
	rds *redis.Redis,
	profileKey string,
	eventID string,
	refreshLastActive bool,
	now time.Time,
) error {
	if rds == nil || profileKey == "" {
		return nil
	}

	raw, err := rds.HgetallCtx(ctx, profileKey)
	if err != nil {
		return err
	}
	var positiveCount int
	var negativeCount int
	for _, weight := range parseWeights(raw) {
		if weight > 0 {
			positiveCount++
			continue
		}
		negativeCount++
	}

	fields := map[string]string{
		"_tag_count":      strconv.Itoa(positiveCount + negativeCount),
		"_positive_count": strconv.Itoa(positiveCount),
		"_negative_count": strconv.Itoa(negativeCount),
	}
	if refreshLastActive {
		fields["_last_active_at"] = strconv.FormatInt(now.Unix(), 10)
	}
	if eventID != "" {
		fields["_last_event_id"] = eventID
	}
	for field, value := range fields {
		if err := rds.HsetCtx(ctx, profileKey, field, value); err != nil {
			return err
		}
	}

	_, err = rds.HincrbyCtx(ctx, profileKey, "_profile_version", 1)
	return err
}
