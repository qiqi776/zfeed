package recommend

import (
	"context"
	"strconv"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

const (
	seenWindow       = 24 * time.Hour
	seenHistorySlack = time.Hour
)

func RecordSeenContents(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	userID int64,
	contentIDs []int64,
	now time.Time,
) error {
	if rds == nil || userID <= 0 || len(contentIDs) == 0 {
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	cfg = NormalizeConfig(cfg)

	pairs := make([]gzredis.Pair, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		pairs = append(pairs, gzredis.Pair{
			Key:   strconv.FormatInt(contentID, 10),
			Score: now.Unix(),
		})
	}
	if len(pairs) == 0 {
		return nil
	}

	key := redisconsts.BuildRecommendSeenKey(userID)
	if _, err := rds.ZaddsCtx(ctx, key, pairs...); err != nil {
		return err
	}
	for i, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		contentKey := strconv.FormatInt(contentID, 10)
		historyKey := redisconsts.BuildRecommendSeenHistoryKey(userID, contentID)
		if _, err := rds.ZaddCtx(ctx, historyKey, now.Unix(), seenHistoryMember(now, i)); err != nil {
			return err
		}
		if _, err := rds.HincrbyCtx(ctx, redisconsts.BuildRecommendSeenCountKey(userID), contentKey, 1); err != nil {
			return err
		}
	}
	if err := rds.ExpireCtx(ctx, key, cfg.SeenTTL); err != nil {
		return err
	}
	if err := rds.ExpireCtx(ctx, redisconsts.BuildRecommendSeenCountKey(userID), cfg.SeenTTL); err != nil {
		return err
	}
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		historyKey := redisconsts.BuildRecommendSeenHistoryKey(userID, contentID)
		if err := rds.ExpireCtx(ctx, historyKey, seenHistoryTTL(cfg.SeenTTL)); err != nil {
			return err
		}
	}
	return nil
}

func LoadSeenCounts(
	ctx context.Context,
	rds *gzredis.Redis,
	userID int64,
	contentIDs []int64,
) (map[int64]int, error) {
	result := make(map[int64]int)
	if rds == nil || userID <= 0 || len(contentIDs) == 0 {
		return result, nil
	}

	fields := make([]string, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		fields = append(fields, strconv.FormatInt(contentID, 10))
	}
	if len(fields) == 0 {
		return result, nil
	}

	now := time.Now()
	windowStart := now.Add(-seenWindow).Unix()
	for _, field := range fields {
		contentID, _ := strconv.ParseInt(field, 10, 64)
		historyKey := redisconsts.BuildRecommendSeenHistoryKey(userID, contentID)
		if _, err := rds.ZremrangebyscoreCtx(ctx, historyKey, 0, windowStart-1); err != nil && !isRedisNil(err) {
			return nil, err
		}
		count, err := rds.ZcountCtx(ctx, historyKey, windowStart, now.Unix())
		if err != nil && !isRedisNil(err) {
			return nil, err
		}
		if count > 0 {
			result[contentID] = count
		}
	}

	key := redisconsts.BuildRecommendSeenKey(userID)
	legacyCounts, err := rds.HmgetCtx(ctx, redisconsts.BuildRecommendSeenCountKey(userID), fields...)
	if err != nil && !isRedisNil(err) {
		return nil, err
	}
	for i, field := range fields {
		contentID, _ := strconv.ParseInt(field, 10, 64)
		if result[contentID] > 0 {
			continue
		}
		if i < len(legacyCounts) && legacyCounts[i] != "" {
			count, parseErr := strconv.Atoi(legacyCounts[i])
			if parseErr != nil {
				return nil, parseErr
			}
			if count > 0 && isRecentSeen(ctx, rds, key, field, now) {
				result[contentID] = count
			}
			continue
		}

		score, err := rds.ZscoreCtx(ctx, key, field)
		if err != nil {
			if isRedisNil(err) {
				continue
			}
			return nil, err
		}
		if score != 0 {
			if score < now.Add(-24*time.Hour).Unix() {
				continue
			}
			result[contentID] = 1
		}
	}
	return result, nil
}

func isRecentSeen(ctx context.Context, rds *gzredis.Redis, key string, field string, now time.Time) bool {
	score, err := rds.ZscoreCtx(ctx, key, field)
	if err != nil {
		return false
	}
	return score >= now.Add(-seenWindow).Unix()
}

func seenHistoryMember(at time.Time, sequence int) string {
	return strconv.FormatInt(at.UnixNano(), 10) + ":" + strconv.Itoa(sequence)
}

func seenHistoryTTL(seenTTL int) int {
	minTTL := int((seenWindow + seenHistorySlack).Seconds())
	if seenTTL < minTTL {
		return minTTL
	}
	return seenTTL
}
