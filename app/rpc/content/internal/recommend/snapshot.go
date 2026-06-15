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
	snapshotBaseRank = 1_000_000
)

type SnapshotMeta struct {
	VariantID  string
	ConfigHash string
}

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
	return SavePersonalizedSnapshotWithMeta(ctx, rds, cfg, userID, candidates, SnapshotMeta{}, now)
}

func SavePersonalizedSnapshotWithMeta(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	userID int64,
	candidates []Candidate,
	meta SnapshotMeta,
	now time.Time,
) (string, error) {
	if rds == nil || len(candidates) == 0 {
		return "", nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	cfg = NormalizeConfig(cfg)
	meta = normalizeSnapshotMeta(meta)

	snapshotID := buildSnapshotID(userID, meta, now)
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
	if err := savePersonalizedSnapshotSources(ctx, rds, cfg, snapshotID, candidates); err != nil {
		return "", err
	}

	metaKey := redisconsts.BuildRecommendUserSnapshotMetaKey(snapshotID)
	if err := rds.HsetCtx(ctx, metaKey, "user_bucket", fmt.Sprintf("%04d", userBucket(userID))); err != nil {
		return "", err
	}
	if err := rds.HsetCtx(ctx, metaKey, "variant_id", meta.VariantID); err != nil {
		return "", err
	}
	if err := rds.HsetCtx(ctx, metaKey, "config_hash", meta.ConfigHash); err != nil {
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

func savePersonalizedSnapshotSources(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	snapshotID string,
	candidates []Candidate,
) error {
	sourceKey := redisconsts.BuildRecommendUserSnapshotSourceKey(snapshotID)
	wrote := false
	for _, candidate := range candidates {
		if candidate.ContentID <= 0 {
			continue
		}

		source := PrimarySource(candidate)
		if source == "" {
			continue
		}
		if err := rds.HsetCtx(
			ctx,
			sourceKey,
			strconv.FormatInt(candidate.ContentID, 10),
			string(source),
		); err != nil {
			return err
		}
		wrote = true
	}
	if !wrote {
		return nil
	}
	return rds.ExpireCtx(ctx, sourceKey, cfg.SnapshotTTL)
}

func LoadPersonalizedSnapshotMeta(
	ctx context.Context,
	rds *gzredis.Redis,
	snapshotID string,
) (SnapshotMeta, bool, error) {
	if rds == nil || !IsPersonalizedSnapshot(snapshotID) {
		return SnapshotMeta{}, false, nil
	}

	raw, err := rds.HgetallCtx(ctx, redisconsts.BuildRecommendUserSnapshotMetaKey(snapshotID))
	if err != nil {
		return SnapshotMeta{}, false, err
	}
	if len(raw) == 0 {
		return SnapshotMeta{}, false, nil
	}

	return normalizeSnapshotMeta(SnapshotMeta{
		VariantID:  raw["variant_id"],
		ConfigHash: raw["config_hash"],
	}), true, nil
}

func LoadPersonalizedSnapshotSources(
	ctx context.Context,
	rds *gzredis.Redis,
	snapshotID string,
	contentIDs []int64,
) (map[int64]Source, error) {
	if rds == nil || !IsPersonalizedSnapshot(snapshotID) || len(contentIDs) == 0 {
		return map[int64]Source{}, nil
	}

	sourceKey := redisconsts.BuildRecommendUserSnapshotSourceKey(snapshotID)
	result := make(map[int64]Source, len(contentIDs))
	rawSources, err := rds.HgetallCtx(ctx, sourceKey)
	if err != nil {
		return nil, err
	}
	if len(rawSources) == 0 {
		return result, nil
	}

	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}

		source := normalizeSource(rawSources[strconv.FormatInt(contentID, 10)])
		if source == "" {
			continue
		}
		result[contentID] = source
	}
	return result, nil
}

func normalizeSnapshotMeta(meta SnapshotMeta) SnapshotMeta {
	meta.VariantID = strings.TrimSpace(meta.VariantID)
	if meta.VariantID == "" {
		meta.VariantID = "control"
	}
	meta.ConfigHash = strings.TrimSpace(meta.ConfigHash)
	if meta.ConfigHash == "" {
		meta.ConfigHash = "default"
	}
	return meta
}

func buildSnapshotID(userID int64, meta SnapshotMeta, now time.Time) string {
	meta = normalizeSnapshotMeta(meta)
	return fmt.Sprintf(
		"rec:%04d:%s:%s:%d",
		userBucket(userID),
		meta.VariantID,
		meta.ConfigHash,
		now.UnixNano(),
	)
}

func userBucket(userID int64) int64 {
	if userID < 0 {
		userID = -userID
	}
	return userID % 10000
}
