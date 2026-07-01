package redisx

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	zrangeRevWithScoresByFloatScript = `
return redis.call('ZRANGE', KEYS[1], ARGV[1], ARGV[2], 'REV', 'WITHSCORES')
`
	zrangeByScoreWithScoresAndLimitScript = `
return redis.call('ZRANGE', KEYS[1], ARGV[1], ARGV[2], 'BYSCORE', 'WITHSCORES', 'LIMIT', ARGV[3], ARGV[4])
`
	zrangeRevByScoreWithScoresAndLimitScript = `
return redis.call('ZRANGE', KEYS[1], ARGV[1], ARGV[2], 'REV', 'BYSCORE', 'WITHSCORES', 'LIMIT', ARGV[3], ARGV[4])
`
)

func ZRangeRevWithScoresByFloatCtx(ctx context.Context, rds *redis.Redis, key string, start, stop int64) ([]redis.FloatPair, error) {
	if rds == nil || key == "" {
		return nil, nil
	}

	result, err := rds.EvalCtx(
		ctx,
		zrangeRevWithScoresByFloatScript,
		[]string{key},
		strconv.FormatInt(start, 10),
		strconv.FormatInt(stop, 10),
	)
	if err != nil {
		return nil, err
	}
	return parseFloatPairs(result)
}

func ZRangeByScoreWithScoresAndLimitCtx(ctx context.Context, rds *redis.Redis, key, minScore, maxScore string, offset, count int) ([]redis.Pair, error) {
	if rds == nil || key == "" || count <= 0 {
		return nil, nil
	}

	result, err := rds.EvalCtx(
		ctx,
		zrangeByScoreWithScoresAndLimitScript,
		[]string{key},
		minScore,
		maxScore,
		strconv.Itoa(offset),
		strconv.Itoa(count),
	)
	if err != nil {
		return nil, err
	}
	return parsePairs(result)
}

func ZRangeRevByScoreWithScoresAndLimitCtx(ctx context.Context, rds *redis.Redis, key, maxScore, minScore string, offset, count int) ([]redis.Pair, error) {
	if rds == nil || key == "" || count <= 0 {
		return nil, nil
	}

	result, err := rds.EvalCtx(
		ctx,
		zrangeRevByScoreWithScoresAndLimitScript,
		[]string{key},
		maxScore,
		minScore,
		strconv.Itoa(offset),
		strconv.Itoa(count),
	)
	if err != nil {
		return nil, err
	}
	return parsePairs(result)
}

func parsePairs(raw any) ([]redis.Pair, error) {
	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return nil, nil
	}

	pairs := make([]redis.Pair, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		key, ok := asString(values[i])
		if !ok {
			return nil, fmt.Errorf("redis zrange member at index %d has type %T", i, values[i])
		}
		score, err := asInt64(values[i+1])
		if err != nil {
			return nil, fmt.Errorf("redis zrange score for member %q: %w", key, err)
		}
		pairs = append(pairs, redis.Pair{
			Key:   key,
			Score: score,
		})
	}
	return pairs, nil
}

func parseFloatPairs(raw any) ([]redis.FloatPair, error) {
	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return nil, nil
	}

	pairs := make([]redis.FloatPair, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		key, ok := asString(values[i])
		if !ok {
			return nil, fmt.Errorf("redis zrange member at index %d has type %T", i, values[i])
		}
		score, err := asFloat64(values[i+1])
		if err != nil {
			return nil, fmt.Errorf("redis zrange score for member %q: %w", key, err)
		}
		pairs = append(pairs, redis.FloatPair{
			Key:   key,
			Score: score,
		})
	}
	return pairs, nil
}

func asString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		return "", false
	}
}

func asInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected type %T", value)
	}
}

func asFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case []byte:
		return strconv.ParseFloat(string(v), 64)
	default:
		return 0, fmt.Errorf("unexpected type %T", value)
	}
}
