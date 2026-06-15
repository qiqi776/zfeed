package recommend

import (
	"context"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

const (
	ActionLike     = "like"
	ActionFavorite = "favorite"
	ActionComment  = "comment"
	ActionClick    = "click"
	ActionDwell    = "dwell"
	ActionUnlike   = "unlike"
)

type ProfileEvent struct {
	EventID   string
	EventType string
	UserID    int64
	ContentID int64
	Weight    float64
}

func ApplyProfileEvent(ctx context.Context, rds *redis.Redis, cfg contentconfig.RecommendConfig, event ProfileEvent) error {
	if rds == nil || event.UserID <= 0 || event.ContentID <= 0 {
		return nil
	}
	cfg = NormalizeConfig(cfg)
	if event.EventID != "" {
		ok, err := rds.SetnxExCtx(ctx, "rec:profile:event:"+event.EventID, "1", 24*3600)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
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
	for tag, weight := range tags {
		currentRaw, err := rds.HgetCtx(ctx, profileKey, tag)
		if err != nil && !isRedisNil(err) {
			return err
		}
		current, _ := strconv.ParseFloat(currentRaw, 64)
		next := current + delta*weight
		if err := rds.HsetCtx(ctx, profileKey, tag, strconv.FormatFloat(next, 'f', 6, 64)); err != nil {
			return err
		}
	}
	if err := rds.HsetCtx(ctx, profileKey, "_updated_at", strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
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
	tags := parseWeights(raw)
	if len(tags) < cfg.MinTags {
		return map[string]float64{}, nil
	}
	return topWeights(tags, cfg.TopTags), nil
}

func ProfileActionForEventType(eventType string) (string, bool) {
	switch eventType {
	case ActionClick,
		ActionDwell,
		ActionLike,
		ActionFavorite,
		ActionComment,
		ActionUnlike:
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
	default:
		return 0
	}
}
