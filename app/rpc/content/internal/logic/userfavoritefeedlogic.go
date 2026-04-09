package logic

import (
	"context"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/svc"
	favoriteservice "zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/pkg/errorx"
)

const (
	userFavoriteFeedKeepN = redisconsts.RedisUserFavoriteKeepLatestN

	userFavoriteFeedRebuildLockTTLSeconds = 30
	userFavoriteFeedRebuildRetryTimes     = 3
	userFavoriteFeedRebuildRetryInterval  = 80 * time.Millisecond
)

type UserFavoriteFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	itemBuilder *FeedItemBuilder
}

func NewUserFavoriteFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFavoriteFeedLogic {
	return &UserFavoriteFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		itemBuilder: NewFeedItemBuilder(ctx, svcCtx),
	}
}

func (l *UserFavoriteFeedLogic) UserFavoriteFeed(in *contentpb.UserFavoriteFeedReq) (*contentpb.UserFavoriteFeedRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	feedKey := redisconsts.BuildUserFavoriteFeedKey(in.GetUserId())
	ids, nextCursor, hasMore, err := l.loadPageIDs(feedKey, in.GetUserId(), in.GetCursor(), pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	contents, err := l.itemBuilder.LoadContentsByIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	items, err := l.itemBuilder.BuildContentItems(contents, in.ViewerId)
	if err != nil {
		return nil, err
	}

	return &contentpb.UserFavoriteFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyUserFavoriteFeedRes() *contentpb.UserFavoriteFeedRes {
	return &contentpb.UserFavoriteFeedRes{
		Items:      []*contentpb.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

func (l *UserFavoriteFeedLogic) loadPageIDs(feedKey string, userID int64, cursor string, pageSize int) ([]int64, string, bool, error) {
	ids, nextCursor, hasMore, cacheExists, err := l.queryUserFavoriteIDs(feedKey, cursor, pageSize)
	if err != nil {
		return nil, "", false, err
	}
	if cacheExists {
		return ids, nextCursor, hasMore, nil
	}

	lockKey := redisconsts.BuildUserFavoriteRebuildLockKey(userID)
	rebuildLock := gzredis.NewRedisLock(l.svcCtx.Redis, lockKey)
	rebuildLock.SetExpire(userFavoriteFeedRebuildLockTTLSeconds)
	locked, lockErr := rebuildLock.AcquireCtx(l.ctx)
	if lockErr != nil {
		return nil, "", false, errorx.Wrap(l.ctx, lockErr, errorx.NewMsg("查询失败请稍后重试"))
	}
	if locked {
		defer func() {
			if releaseOK, releaseErr := rebuildLock.ReleaseCtx(context.Background()); !releaseOK || releaseErr != nil {
				l.Errorf("release user favorite feed rebuild lock failed, key=%s, err=%v", lockKey, releaseErr)
			}
		}()

		allRows, err := l.listAllFavorites(userID)
		if err != nil {
			return nil, "", false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏列表失败"))
		}
		if len(allRows) == 0 {
			return nil, "", false, nil
		}
		if err := l.updateUserFavoriteCache(feedKey, allRows); err != nil {
			l.Errorf("rebuild user favorite feed cache failed, user_id=%d, err=%v", userID, err)
		}

		pageRows := l.pageFavoriteRows(allRows, cursor, pageSize)
		if len(pageRows) > pageSize {
			hasMore = true
			nextCursor = strconv.FormatInt(pageRows[pageSize-1].FavoriteId, 10)
			pageRows = pageRows[:pageSize]
		} else {
			hasMore = false
			nextCursor = ""
		}

		result := make([]int64, 0, len(pageRows))
		for _, row := range pageRows {
			if row == nil || row.GetContentId() <= 0 {
				continue
			}
			result = append(result, row.GetContentId())
		}
		return result, nextCursor, hasMore, nil
	}

	for i := 0; i < userFavoriteFeedRebuildRetryTimes; i++ {
		time.Sleep(userFavoriteFeedRebuildRetryInterval)
		ids, nextCursor, hasMore, cacheExists, err = l.queryUserFavoriteIDs(feedKey, cursor, pageSize)
		if err != nil {
			return nil, "", false, err
		}
		if cacheExists {
			return ids, nextCursor, hasMore, nil
		}
	}

	return nil, "", false, errorx.NewMsg("查询失败请稍后重试")
}

func (l *UserFavoriteFeedLogic) queryUserFavoriteIDs(feedKey, cursor string, pageSize int) ([]int64, string, bool, bool, error) {
	result, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserFavoriteZSetScript,
		[]string{feedKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, "", false, false, errorx.NewMsg("查询收藏列表失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	cacheExists := existsVal == 1
	if !cacheExists {
		return nil, "", false, false, nil
	}

	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1
	nextCursor := ""
	if hasMore {
		if s, ok := luaReplyString(arr[2]); ok {
			nextCursor = s
		}
	}

	ids := make([]int64, 0, len(arr)-3)
	for i := 3; i < len(arr); i++ {
		s, _ := luaReplyString(arr[i])
		if s == "" {
			continue
		}
		id, parseErr := strconv.ParseInt(s, 10, 64)
		if parseErr != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nextCursor, hasMore, true, nil
}

func (l *UserFavoriteFeedLogic) listAllFavorites(userID int64) ([]*favoriteservice.FavoriteItem, error) {
	if l.svcCtx.FavoriteRpc == nil {
		return nil, errorx.NewMsg("查询失败请稍后重试")
	}

	cursor := int64(0)
	pageSize := uint32(500)
	result := make([]*favoriteservice.FavoriteItem, 0)

	for {
		resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteList(l.ctx, &favoriteservice.QueryFavoriteListReq{
			UserId:   userID,
			Cursor:   cursor,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}
		if resp != nil && len(resp.GetItems()) > 0 {
			result = append(result, resp.GetItems()...)
		}
		if resp == nil || !resp.GetHasMore() || resp.GetNextCursor() <= 0 {
			break
		}
		cursor = resp.GetNextCursor()
		if len(result) >= userFavoriteFeedKeepN {
			break
		}
	}

	if len(result) > userFavoriteFeedKeepN {
		result = result[:userFavoriteFeedKeepN]
	}
	return result, nil
}

func (l *UserFavoriteFeedLogic) updateUserFavoriteCache(feedKey string, rows []*favoriteservice.FavoriteItem) error {
	if len(rows) == 0 {
		return nil
	}

	args := make([]any, 0, 1+len(rows)*2)
	args = append(args, strconv.Itoa(userFavoriteFeedKeepN))
	for _, row := range rows {
		if row == nil || row.GetFavoriteId() <= 0 || row.GetContentId() <= 0 {
			continue
		}
		score := strconv.FormatInt(row.GetFavoriteId(), 10)
		member := strconv.FormatInt(row.GetContentId(), 10)
		args = append(args, score, member)
	}

	_, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

func (l *UserFavoriteFeedLogic) pageFavoriteRows(allRows []*favoriteservice.FavoriteItem, cursor string, pageSize int) []*favoriteservice.FavoriteItem {
	if len(allRows) == 0 {
		return allRows
	}

	cursorScore := int64(0)
	if cursor != "" {
		v, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil && v > 0 {
			cursorScore = v
		}
	}

	result := make([]*favoriteservice.FavoriteItem, 0, pageSize+1)
	for _, row := range allRows {
		if row == nil {
			continue
		}
		if cursorScore > 0 && row.GetFavoriteId() >= cursorScore {
			continue
		}
		result = append(result, row)
		if len(result) >= pageSize+1 {
			break
		}
	}
	return result
}
