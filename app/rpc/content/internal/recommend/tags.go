package recommend

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
)

func BasicContentTags(contentType int32, title string, description string) map[string]float64 {
	tags := make(map[string]float64)
	switch contentType {
	case 10:
		tags["type:article"] = 1
	case 20:
		tags["type:video"] = 1
	default:
		tags["type:unknown"] = 0.3
	}

	for _, token := range tokenize(title + " " + description) {
		if _, exists := tags[token]; exists {
			tags[token] += 0.2
			continue
		}
		tags[token] = 0.6
		if len(tags) >= 16 {
			break
		}
	}
	return tags
}

func WriteContentTags(
	ctx context.Context,
	rds *gzredis.Redis,
	cfg contentconfig.RecommendConfig,
	contentID int64,
	tags map[string]float64,
	indexScore float64,
) error {
	if rds == nil || contentID <= 0 || len(tags) == 0 {
		return nil
	}
	cfg = NormalizeConfig(cfg)

	contentKey := redisconsts.BuildRecommendContentTagsKey(contentID)
	member := strconv.FormatInt(contentID, 10)
	for tag, weight := range tags {
		tag = normalizeTag(tag)
		if tag == "" || weight == 0 {
			continue
		}
		if err := rds.HsetCtx(ctx, contentKey, tag, strconv.FormatFloat(weight, 'f', 6, 64)); err != nil {
			return err
		}
		if _, err := rds.ZaddFloatCtx(
			ctx,
			redisconsts.BuildRecommendTagIndexKey(tag),
			indexScore*weight,
			member,
		); err != nil {
			return err
		}
		if err := rds.ExpireCtx(ctx, redisconsts.BuildRecommendTagIndexKey(tag), cfg.Interest.TagIndexTTL); err != nil {
			return err
		}
	}
	return rds.ExpireCtx(ctx, contentKey, cfg.Interest.ContentTagTTL)
}

func LoadContentTags(ctx context.Context, rds *gzredis.Redis, contentID int64) (map[string]float64, error) {
	if rds == nil || contentID <= 0 {
		return map[string]float64{}, nil
	}
	raw, err := rds.HgetallCtx(ctx, redisconsts.BuildRecommendContentTagsKey(contentID))
	if err != nil {
		return nil, err
	}
	return parseWeights(raw), nil
}

func parseWeights(raw map[string]string) map[string]float64 {
	result := make(map[string]float64, len(raw))
	for tag, value := range raw {
		tag = normalizeTag(tag)
		if tag == "" || strings.HasPrefix(tag, "_") {
			continue
		}
		weight, err := strconv.ParseFloat(value, 64)
		if err != nil || weight == 0 {
			continue
		}
		result[tag] = weight
	}
	return result
}

func topWeights(weights map[string]float64, limit int) map[string]float64 {
	if limit <= 0 || len(weights) <= limit {
		result := make(map[string]float64, len(weights))
		for key, value := range weights {
			result[key] = value
		}
		return result
	}

	type pair struct {
		key   string
		value float64
	}
	pairs := make([]pair, 0, len(weights))
	for key, value := range weights {
		pairs = append(pairs, pair{key: key, value: value})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].value == pairs[j].value {
			return pairs[i].key < pairs[j].key
		}
		return pairs[i].value > pairs[j].value
	})

	result := make(map[string]float64, limit)
	for _, item := range pairs[:limit] {
		result[item.key] = item.value
	}
	return result
}

func normalizeTag(tag string) string {
	return strings.TrimSpace(strings.ToLower(tag))
}

func isRedisNil(err error) bool {
	return err == redis.Nil
}

func tokenize(text string) []string {
	replacer := strings.NewReplacer(
		",", " ",
		".", " ",
		";", " ",
		":", " ",
		"!", " ",
		"?", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"\n", " ",
		"\t", " ",
	)
	parts := strings.Fields(replacer.Replace(strings.ToLower(text)))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeTag(part)
		if len(part) < 2 {
			continue
		}
		result = append(result, part)
	}
	return result
}
