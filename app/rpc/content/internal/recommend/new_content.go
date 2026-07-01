package recommend

import (
	"context"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/pkg/redisx"
)

type PublishedContent struct {
	ContentID   int64
	AuthorID    int64
	ContentType int32
	PublishedAt time.Time
	Public      bool
}

func WriteNewContent(ctx context.Context, rds *redis.Redis, cfg contentconfig.RecommendConfig, content PublishedContent) error {
	if rds == nil || content.ContentID <= 0 || !content.Public {
		return nil
	}
	cfg = NormalizeConfig(cfg)

	member := strconv.FormatInt(content.ContentID, 10)
	score := content.PublishedAt.Unix()
	if score <= 0 {
		score = time.Now().Unix()
	}
	if _, err := rds.ZaddCtx(ctx, redisconsts.RecommendNewContentKey, score, member); err != nil {
		return err
	}

	metaKey := redisconsts.BuildRecommendNewContentMetaKey(content.ContentID)
	if err := rds.HsetCtx(ctx, metaKey, "author_id", strconv.FormatInt(content.AuthorID, 10)); err != nil {
		return err
	}
	if err := rds.HsetCtx(ctx, metaKey, "content_type", strconv.FormatInt(int64(content.ContentType), 10)); err != nil {
		return err
	}
	if err := rds.HsetCtx(ctx, metaKey, "published_at", strconv.FormatInt(score, 10)); err != nil {
		return err
	}
	return rds.ExpireCtx(ctx, metaKey, cfg.ColdStartMetaTTL)
}

func RecallNewContent(ctx context.Context, rds *redis.Redis, limit int) ([]int64, error) {
	if rds == nil || limit <= 0 {
		return []int64{}, nil
	}

	pairs, err := redisx.ZRangeRevWithScoresByFloatCtx(ctx, rds, redisconsts.RecommendNewContentKey, 0, int64(limit-1))
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(pairs))
	for _, pair := range pairs {
		id, err := strconv.ParseInt(pair.Key, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}
