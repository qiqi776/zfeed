package logic

import (
	"context"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
)

type CacheResult int

const (
	cacheHit CacheResult = iota
	cacheMiss
	cacheError
)

type hotFeedResult struct {
	ids                []int64
	nextCursor         int64
	hasMore            bool
	resolvedSnapshotID string
}

type RecommendFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	itemBuilder *FeedItemBuilder
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		itemBuilder: NewFeedItemBuilder(ctx, svcCtx),
	}
}

func (l *RecommendFeedLogic) RecommendFeed(in *contentpb.RecommendFeedReq) (*contentpb.RecommendFeedRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	preferredKey, preferredSnapshotID := l.resolveSnapshotKey(in.SnapshotId)
	result, err := l.queryHotIDsByCursor(preferredKey, preferredSnapshotID, strings.TrimSpace(in.GetCursor()), pageSize)
	if err != nil {
		return nil, err
	}
	if len(result.ids) == 0 {
		return &contentpb.RecommendFeedRes{
			Items:      []*contentpb.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: result.resolvedSnapshotID,
		}, nil
	}

	contents, err := l.itemBuilder.LoadContentsByIDs(result.ids)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return &contentpb.RecommendFeedRes{
			Items:      []*contentpb.ContentItem{},
			NextCursor: 0,
			HasMore:    false,
			SnapshotId: result.resolvedSnapshotID,
		}, nil
	}

	items, err := l.itemBuilder.BuildContentItems(contents, in.UserId)
	if err != nil {
		return nil, err
	}

	return &contentpb.RecommendFeedRes{
		Items:      items,
		NextCursor: result.nextCursor,
		HasMore:    result.hasMore,
		SnapshotId: result.resolvedSnapshotID,
	}, nil
}

func (l *RecommendFeedLogic) resolveSnapshotKey(reqSnapshotID *string) (string, string) {
	if reqSnapshotID == nil {
		return "", ""
	}
	snapshotID := strings.TrimSpace(*reqSnapshotID)
	if snapshotID == "" {
		return "", ""
	}
	return redisconsts.BuildHotFeedSnapshotKey(snapshotID), snapshotID
}

func (l *RecommendFeedLogic) queryHotIDsByCursor(preferredKey, preferredSnapshotID, cursor string, pageSize int) (*hotFeedResult, error) {
	result, cacheResult := l.queryFromRedis(preferredKey, preferredSnapshotID, cursor, pageSize)
	if cacheResult == cacheHit {
		return result, nil
	}
	return nil, mapHotFeedCacheError(cacheResult)
}

func mapHotFeedCacheError(cacheResult CacheResult) error {
	switch cacheResult {
	case cacheMiss:
		return errorx.NewMsg("热榜缓存不存在")
	case cacheError:
		return errorx.NewMsg("查询热榜索引失败")
	default:
		return errorx.NewMsg("查询失败请稍后重试")
	}
}

func (l *RecommendFeedLogic) queryFromRedis(preferredKey, preferredSnapshotID, cursor string, pageSize int) (*hotFeedResult, CacheResult) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryHotFeedZSetScript,
		[]string{
			preferredKey,
			redisconsts.RedisFeedHotGlobalLatestKey,
			redisconsts.RedisFeedHotGlobalSnapshotPrefix,
			redisconsts.RedisFeedHotGlobalKey,
		},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
		preferredSnapshotID,
	)
	if err != nil {
		l.Errorf("query hot feed from redis failed: %v", err)
		return nil, cacheError
	}

	parsed, exists, parseErr := parseHotFeedLuaResult(res)
	if parseErr != nil {
		l.Errorf("parse hot feed lua result failed: %v", parseErr)
		return nil, cacheError
	}
	if !exists {
		return nil, cacheMiss
	}
	return parsed, cacheHit
}

func parseHotFeedLuaResult(res any) (*hotFeedResult, bool, error) {
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 4 {
		return nil, false, errorx.NewMsg("查询热榜索引失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	exists := existsVal == 1
	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1

	nextCursor := int64(0)
	if hasMore {
		nextStr, _ := luaReplyString(arr[2])
		if nextStr != "" {
			value, err := strconv.ParseInt(nextStr, 10, 64)
			if err != nil {
				return nil, false, errorx.NewMsg("查询热榜索引失败")
			}
			nextCursor = value
		}
	}

	resolvedSnapshotID, _ := luaReplyString(arr[3])

	ids := make([]int64, 0, len(arr)-4)
	for i := 4; i < len(arr); i++ {
		idStr, _ := luaReplyString(arr[i])
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}

	return &hotFeedResult{
		ids:                ids,
		nextCursor:         nextCursor,
		hasMore:            hasMore,
		resolvedSnapshotID: resolvedSnapshotID,
	}, exists, nil
}
