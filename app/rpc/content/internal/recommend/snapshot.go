package recommend

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

const (
	snapshotPrefix   = "rec:"
	snapshotVariant  = "control"
	snapshotConfigID = "default"
	snapshotBaseRank = 1_000_000
)

func IsPersonalizedSnapshot(snapshotID string) bool {
	return strings.HasPrefix(strings.TrimSpace(snapshotID), snapshotPrefix)
}

func SavePersonalizedSnapshot(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	userID int64,
	candidates []Candidate,
	now time.Time,
) (string, error) {
	if rds == nil || len(candidates) == 0 {
		return "", nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	cfg = NormalizeConfig(cfg)

	snapshotID := buildSnapshotID(userID, now)
	snapshotKey := redisconsts.BuildRecommendUserSnapshotKey(snapshotID)
	pairs := make([]gzredis.Pair, 0, len(candidates))
	for rank, candidate := range candidates {
		if candidate.ContentID <= 0 {
			continue
		}
		pairs = append(pairs, gzredis.Pair{
			Key:   strconv.FormatInt(candidate.ContentID, 10),
			Score: int64(snapshotBaseRank - rank),
		})
	}
	if len(pairs) == 0 {
		return "", nil
	}
	if _, err := rds.ZaddsCtx(ctx, snapshotKey, pairs...); err != nil {
		return "", err
	}
	if err := rds.ExpireCtx(ctx, snapshotKey, cfg.SnapshotTTL); err != nil {
		return "", err
	}

	metaKey := redisconsts.BuildRecommendUserSnapshotMetaKey(snapshotID)
	if err := rds.HsetCtx(ctx, metaKey, "user_bucket", fmt.Sprintf("%04d", userBucket(userID))); err != nil {
		return "", err
	}
	if err := rds.HsetCtx(ctx, metaKey, "variant_id", snapshotVariant); err != nil {
		return "", err
	}
	if err := rds.HsetCtx(ctx, metaKey, "config_hash", snapshotConfigID); err != nil {
		return "", err
	}
	if err := rds.HsetCtx(ctx, metaKey, "created_at", strconv.FormatInt(now.Unix(), 10)); err != nil {
		return "", err
	}
	if err := rds.HsetCtx(ctx, metaKey, "recall_size", strconv.Itoa(len(pairs))); err != nil {
		return "", err
	}
	if err := rds.ExpireCtx(ctx, metaKey, cfg.SnapshotTTL); err != nil {
		return "", err
	}
	return snapshotID, nil
}

func buildSnapshotID(userID int64, now time.Time) string {
	return fmt.Sprintf(
		"rec:%04d:%s:%s:%d",
		userBucket(userID),
		snapshotVariant,
		snapshotConfigID,
		now.UnixNano(),
	)
}

func userBucket(userID int64) int64 {
	if userID < 0 {
		userID = -userID
	}
	return userID % 10000
}
