package commentservicelogic

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	goredis "github.com/zeromicro/go-zero/core/stores/redis"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	luautils "zfeed/app/rpc/interaction/internal/common/utils/lua"
)

const (
	commentRebuildRetryTimes = 3
	commentRebuildRetryDelay = 50 * time.Millisecond
)

func readCachedCommentIndexIDs(ctx context.Context, rds *goredis.Redis, key string, cursor int64, pageSize uint32) ([]int64, bool, error) {
	if rds == nil || key == "" {
		return nil, false, nil
	}

	exists, err := rds.ExistsCtx(ctx, key)
	if err != nil || !exists {
		return nil, exists, err
	}

	upper := maxCommentCacheScore(cursor)
	pairs, err := rds.ZrevrangebyscoreWithScoresAndLimitCtx(ctx, key, 0, upper, 0, int(pageSize)+1)
	if err != nil {
		return nil, true, err
	}

	ids := make([]int64, 0, len(pairs))
	for _, pair := range pairs {
		commentID, convErr := strconv.ParseInt(pair.Key, 10, 64)
		if convErr != nil || commentID <= 0 {
			continue
		}
		ids = append(ids, commentID)
	}
	return ids, true, nil
}

func readCachedCommentItems(ctx context.Context, rds *goredis.Redis, commentIDs []int64) (map[int64]*interaction.CommentItem, []int64, error) {
	result := make(map[int64]*interaction.CommentItem, len(commentIDs))
	ids := uniqueCommentIDs(commentIDs)
	if rds == nil || len(ids) == 0 {
		return result, ids, nil
	}

	keys := make([]string, 0, len(ids))
	for _, commentID := range ids {
		keys = append(keys, rediskey.BuildCommentItemKey(strconv.FormatInt(commentID, 10)))
	}

	resp, err := rds.EvalCtx(ctx, luautils.BatchGetCommentObjsScript, keys)
	if err != nil {
		return nil, ids, err
	}

	values, ok := resp.([]any)
	if !ok {
		return nil, ids, nil
	}

	missIDs := make([]int64, 0)
	for idx, raw := range values {
		commentID := ids[idx]
		payload, ok := stringifyLuaValue(raw)
		if !ok || payload == "" {
			missIDs = append(missIDs, commentID)
			continue
		}

		item, err := unmarshalCommentItem(payload)
		if err != nil {
			missIDs = append(missIDs, commentID)
			continue
		}

		result[commentID] = item
	}

	return result, missIDs, nil
}

func cacheCommentItemsBestEffort(ctx context.Context, logger logx.Logger, rds *goredis.Redis, items []*interaction.CommentItem) {
	if rds == nil {
		return
	}

	for _, item := range items {
		if item == nil || item.GetCommentId() <= 0 {
			continue
		}

		payload, err := marshalCommentItem(item)
		if err != nil {
			logger.Errorf("序列化评论缓存失败: %v, comment_id=%d", err, item.GetCommentId())
			continue
		}

		key := rediskey.BuildCommentItemKey(strconv.FormatInt(item.GetCommentId(), 10))
		if err := rds.SetexCtx(ctx, key, string(payload), rediskey.RedisCommentItemExpireSecs); err != nil {
			logger.Errorf("写评论对象缓存失败: %v, comment_id=%d", err, item.GetCommentId())
		}
	}
}

func cacheCommentItemsAndIndexBestEffort(ctx context.Context, logger logx.Logger, rds *goredis.Redis, key string, items []*interaction.CommentItem) {
	if rds == nil || key == "" {
		return
	}

	for _, item := range items {
		if item == nil || item.GetCommentId() <= 0 {
			continue
		}

		payload, err := marshalCommentItem(item)
		if err != nil {
			logger.Errorf("序列化评论缓存失败: %v, comment_id=%d", err, item.GetCommentId())
			continue
		}

		objKey := rediskey.BuildCommentItemKey(strconv.FormatInt(item.GetCommentId(), 10))
		if _, err := rds.EvalCtx(
			ctx,
			luautils.UpdateCommentCacheScript,
			[]string{objKey, key},
			strconv.Itoa(rediskey.RedisCommentItemExpireSecs),
			strconv.Itoa(rediskey.RedisCommentIndexExpireSecs),
			strconv.FormatInt(item.GetCommentId(), 10),
			payload,
		); err != nil {
			logger.Errorf("原子更新评论缓存失败: %v, key=%s, comment_id=%d", err, key, item.GetCommentId())
		}
	}
}

func invalidateCommentCacheKeysBestEffort(ctx context.Context, logger logx.Logger, rds *goredis.Redis, keys ...string) {
	if rds == nil {
		return
	}

	uniq := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		uniq = append(uniq, key)
	}

	if len(uniq) == 0 {
		return
	}

	if _, err := rds.DelCtx(ctx, uniq...); err != nil {
		logger.Errorf("删除评论缓存失败: %v, keys=%v", err, uniq)
	}
}

func buildCachedCommentResult(ids []int64, itemMap map[int64]*interaction.CommentItem, pageSize uint32) ([]*interaction.CommentItem, int64, bool, bool) {
	if len(ids) == 0 {
		return []*interaction.CommentItem{}, 0, false, true
	}

	items := make([]*interaction.CommentItem, 0, len(ids))
	for _, commentID := range ids {
		item := itemMap[commentID]
		if item == nil {
			return nil, 0, false, false
		}
		items = append(items, item)
	}

	if uint32(len(items)) <= pageSize {
		return items, 0, false, true
	}

	trimmed := items[:pageSize]
	return trimmed, trimmed[len(trimmed)-1].GetCommentId(), true, true
}

func marshalCommentItem(item *interaction.CommentItem) (string, error) {
	if item == nil {
		return "", nil
	}

	payload, err := json.Marshal(item)
	if err != nil {
		return "", err
	}

	return string(payload), nil
}

func unmarshalCommentItem(raw string) (*interaction.CommentItem, error) {
	if raw == "" {
		return nil, nil
	}

	var item interaction.CommentItem
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return nil, err
	}

	cloned := item
	return &cloned, nil
}

func stringifyLuaValue(raw any) (string, bool) {
	switch v := raw.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		return "", false
	}
}

func tryAcquireCommentRebuildLock(ctx context.Context, rds *goredis.Redis, key string) (bool, error) {
	if rds == nil || key == "" {
		return false, nil
	}

	token := strconv.FormatInt(time.Now().UnixNano(), 10)
	return rds.SetnxExCtx(ctx, key, token, rediskey.RedisCommentLockExpireSecs)
}

func releaseCommentRebuildLock(ctx context.Context, logger logx.Logger, rds *goredis.Redis, key string) {
	if rds == nil || key == "" {
		return
	}

	if _, err := rds.DelCtx(ctx, key); err != nil {
		logger.Errorf("释放评论重建锁失败: %v, key=%s", err, key)
	}
}

func maxCommentCacheScore(cursor int64) int64 {
	if cursor <= 0 {
		return math.MaxInt64
	}
	if cursor == math.MinInt64 {
		return 0
	}
	return cursor - 1
}
