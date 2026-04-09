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
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
)

const (
	userPublishFeedKeepN = redisconsts.RedisUserPublishKeepLatestN

	userPublishFeedRebuildLockTTLSeconds = 30
	userPublishFeedRebuildRetryTimes     = 3
	userPublishFeedRebuildRetryInterval  = 80 * time.Millisecond
)

type UserPublishFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	itemBuilder *FeedItemBuilder
}

func NewUserPublishFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishFeedLogic {
	return &UserPublishFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		itemBuilder: NewFeedItemBuilder(ctx, svcCtx),
	}
}

func (l *UserPublishFeedLogic) UserPublishFeed(in *contentpb.UserPublishFeedReq) (*contentpb.UserPublishFeedRes, error) {
	if in == nil || in.GetAuthorId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	feedKey := redisconsts.BuildUserPublishFeedKey(in.GetAuthorId())
	ids, nextCursor, hasMore, err := l.loadPageIDs(feedKey, in.GetAuthorId(), in.GetCursor(), pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	contents, err := l.itemBuilder.LoadContentsByIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	items, err := l.itemBuilder.BuildContentItems(contents, in.ViewerId)
	if err != nil {
		return nil, err
	}

	return &contentpb.UserPublishFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyUserPublishFeedRes() *contentpb.UserPublishFeedRes {
	return &contentpb.UserPublishFeedRes{
		Items:      []*contentpb.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

func (l *UserPublishFeedLogic) loadPageIDs(feedKey string, authorID int64, cursor string, pageSize int) ([]int64, string, bool, error) {
	ids, nextCursor, hasMore, cacheExists, err := l.queryUserPublishIDs(feedKey, cursor, pageSize)
	if err != nil {
		return nil, "", false, err
	}
	if cacheExists {
		return ids, nextCursor, hasMore, nil
	}

	lockKey := redisconsts.BuildUserPublishRebuildLockKey(authorID)
	rebuildLock := gzredis.NewRedisLock(l.svcCtx.Redis, lockKey)
	rebuildLock.SetExpire(userPublishFeedRebuildLockTTLSeconds)
	locked, lockErr := rebuildLock.AcquireCtx(l.ctx)
	if lockErr != nil {
		return nil, "", false, errorx.Wrap(l.ctx, lockErr, errorx.NewMsg("查询失败请稍后重试"))
	}
	if locked {
		defer func() {
			if releaseOK, releaseErr := rebuildLock.ReleaseCtx(context.Background()); !releaseOK || releaseErr != nil {
				l.Errorf("release user publish feed rebuild lock failed, key=%s, err=%v", lockKey, releaseErr)
			}
		}()

		allRows, err := l.contentRepo.ListPublishedByAuthor(authorID)
		if err != nil {
			return nil, "", false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布内容失败"))
		}
		if len(allRows) == 0 {
			return nil, "", false, nil
		}
		if err := l.updateUserPublishCache(feedKey, allRows); err != nil {
			l.Errorf("rebuild user publish feed cache failed, author_id=%d, err=%v", authorID, err)
		}

		pageRows := l.pageUserPublishRows(allRows, cursor, pageSize)
		if len(pageRows) > pageSize {
			hasMore = true
			nextCursor = strconv.FormatInt(pageRows[pageSize-1].ID, 10)
			pageRows = pageRows[:pageSize]
		} else {
			hasMore = false
			nextCursor = ""
		}

		result := make([]int64, 0, len(pageRows))
		for _, row := range pageRows {
			if row == nil || row.ID <= 0 {
				continue
			}
			result = append(result, row.ID)
		}
		return result, nextCursor, hasMore, nil
	}

	for i := 0; i < userPublishFeedRebuildRetryTimes; i++ {
		time.Sleep(userPublishFeedRebuildRetryInterval)
		ids, nextCursor, hasMore, cacheExists, err = l.queryUserPublishIDs(feedKey, cursor, pageSize)
		if err != nil {
			return nil, "", false, err
		}
		if cacheExists {
			return ids, nextCursor, hasMore, nil
		}
	}

	return nil, "", false, errorx.NewMsg("查询失败请稍后重试")
}

func (l *UserPublishFeedLogic) queryUserPublishIDs(feedKey, cursor string, pageSize int) ([]int64, string, bool, bool, error) {
	result, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserPublishZSetScript,
		[]string{feedKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布列表失败"))
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, "", false, false, errorx.NewMsg("查询发布列表失败")
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

func (l *UserPublishFeedLogic) updateUserPublishCache(feedKey string, rows []*model.ZfeedContent) error {
	if len(rows) == 0 {
		return nil
	}

	args := make([]any, 0, 1+len(rows)*2)
	args = append(args, strconv.Itoa(userPublishFeedKeepN))
	for _, row := range rows {
		if row == nil || row.ID <= 0 {
			continue
		}
		idStr := strconv.FormatInt(row.ID, 10)
		args = append(args, idStr, idStr)
	}

	_, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

func (l *UserPublishFeedLogic) pageUserPublishRows(allRows []*model.ZfeedContent, cursor string, pageSize int) []*model.ZfeedContent {
	if len(allRows) == 0 {
		return allRows
	}

	cursorID := int64(0)
	if cursor != "" {
		v, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil && v > 0 {
			cursorID = v
		}
	}

	result := make([]*model.ZfeedContent, 0, pageSize+1)
	for _, row := range allRows {
		if row == nil {
			continue
		}
		if cursorID > 0 && row.ID >= cursorID {
			continue
		}
		result = append(result, row)
		if len(result) >= pageSize+1 {
			break
		}
	}
	return result
}
