package logic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	followservice "zfeed/app/rpc/interaction/client/followservice"
	"zfeed/pkg/errorx"
)

const (
	followInboxRebuildLockTTLSeconds = 30
	followInboxKeepN                 = redisconsts.RedisFollowInboxKeepLatestN
	followFeedMaxPageSize            = 50
	followFeedDefaultPageSize        = 10
	followFeedFolloweePageSize       = 500
	followFeedFolloweeLimit          = 5000
)

type FollowFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewFollowFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowFeedLogic {
	return &FollowFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *FollowFeedLogic) FollowFeed(in *contentpb.FollowFeedReq) (*contentpb.FollowFeedRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = followFeedDefaultPageSize
	}
	if pageSize > followFeedMaxPageSize {
		pageSize = followFeedMaxPageSize
	}

	inboxKey := redisconsts.BuildFollowInboxKey(in.GetUserId())
	ids, nextCursor, hasMore, err := l.loadPageIDs(in.GetUserId(), inboxKey, in.GetCursor(), pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return emptyFollowFeedRes(), nil
	}

	contents, err := l.loadContents(ids)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return emptyFollowFeedRes(), nil
	}

	items, err := l.buildFollowItems(contents)
	if err != nil {
		return nil, err
	}

	return &contentpb.FollowFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyFollowFeedRes() *contentpb.FollowFeedRes {
	return &contentpb.FollowFeedRes{
		Items:      []*contentpb.FollowFeedItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

func (l *FollowFeedLogic) loadPageIDs(userID int64, inboxKey, cursor string, pageSize int) ([]int64, string, bool, error) {
	ids, nextCursor, hasMore, cacheExists, err := l.queryInboxIDs(inboxKey, cursor, pageSize)
	if err != nil {
		return nil, "", false, err
	}
	if cacheExists {
		return ids, nextCursor, hasMore, nil
	}

	lockKey := redisconsts.BuildFollowInboxRebuildLockKey(userID)
	rebuildLock := gzredis.NewRedisLock(l.svcCtx.Redis, lockKey)
	rebuildLock.SetExpire(followInboxRebuildLockTTLSeconds)

	locked, lockErr := rebuildLock.AcquireCtx(l.ctx)
	if lockErr != nil {
		return nil, "", false, errorx.Wrap(l.ctx, lockErr, errorx.NewMsg("查询失败请稍后重试"))
	}
	if !locked {
		return nil, "", false, errorx.NewMsg("查询失败请稍后重试")
	}
	defer func() {
		if releaseOK, releaseErr := rebuildLock.ReleaseCtx(context.Background()); !releaseOK || releaseErr != nil {
			l.Errorf("release follow inbox rebuild lock failed, key=%s, err=%v", lockKey, releaseErr)
		}
	}()

	rebuilt, rebuildErr := l.rebuildInboxCacheBestEffort(userID, inboxKey)
	if rebuildErr != nil {
		return nil, "", false, rebuildErr
	}
	if !rebuilt {
		return nil, "", false, nil
	}

	ids, nextCursor, hasMore, cacheExists, err = l.queryInboxIDs(inboxKey, cursor, pageSize)
	if err != nil {
		return nil, "", false, err
	}
	if !cacheExists {
		return nil, "", false, nil
	}
	return ids, nextCursor, hasMore, nil
}

func (l *FollowFeedLogic) queryInboxIDs(inboxKey, cursor string, pageSize int) ([]int64, string, bool, bool, error) {
	result, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryFollowInboxZSetScript,
		[]string{inboxKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注收件箱失败"))
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, "", false, false, errorx.NewMsg("查询关注收件箱失败")
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
		value, _ := luaReplyString(arr[i])
		if value == "" {
			continue
		}
		id, parseErr := strconv.ParseInt(value, 10, 64)
		if parseErr != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nextCursor, hasMore, true, nil
}

func (l *FollowFeedLogic) rebuildInboxCacheBestEffort(userID int64, inboxKey string) (bool, error) {
	followees, err := l.listFollowees(userID)
	if err != nil {
		return false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
	}
	if len(followees) == 0 {
		return false, nil
	}

	rows, err := l.contentRepo.ListFollowByAuthorsCursor(followees, 0, followInboxKeepN)
	if err != nil {
		return false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注内容失败"))
	}
	if len(rows) == 0 {
		return false, nil
	}

	if err := l.updateInboxCache(inboxKey, rows); err != nil {
		return false, err
	}
	return true, nil
}

func (l *FollowFeedLogic) listFollowees(userID int64) ([]int64, error) {
	if l.svcCtx.FollowRpc == nil {
		return nil, errorx.NewMsg("查询失败请稍后重试")
	}

	followees := make([]int64, 0)
	cursor := int64(0)
	for {
		resp, err := l.svcCtx.FollowRpc.ListFollowees(l.ctx, &followservice.ListFolloweesReq{
			UserId:   userID,
			Cursor:   cursor,
			PageSize: followFeedFolloweePageSize,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}
		if len(resp.GetFollowUserIds()) > 0 {
			followees = append(followees, resp.GetFollowUserIds()...)
		}
		if !resp.GetHasMore() || resp.GetNextCursor() <= 0 {
			break
		}
		cursor = resp.GetNextCursor()
		if len(followees) >= followFeedFolloweeLimit {
			break
		}
	}
	return followees, nil
}

func (l *FollowFeedLogic) updateInboxCache(inboxKey string, rows []*model.ZfeedContent) error {
	if len(rows) == 0 {
		return nil
	}

	args := make([]any, 0, 1+len(rows)*2)
	args = append(args, strconv.Itoa(followInboxKeepN))
	for _, row := range rows {
		if row == nil || row.ID <= 0 {
			continue
		}
		contentID := strconv.FormatInt(row.ID, 10)
		args = append(args, contentID, contentID)
	}

	_, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.UpdateFollowInboxZSetScript, []string{inboxKey}, args...)
	if err != nil {
		return errorx.Wrap(l.ctx, err, errorx.NewMsg("回填关注收件箱失败"))
	}
	return nil
}

func (l *FollowFeedLogic) loadContents(ids []int64) ([]*model.ZfeedContent, error) {
	contentMap, err := l.contentRepo.BatchGetPublishedByIDs(ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注内容失败"))
	}

	contents := make([]*model.ZfeedContent, 0, len(ids))
	for _, id := range ids {
		if row, ok := contentMap[id]; ok && row != nil {
			contents = append(contents, row)
		}
	}
	return contents, nil
}

func (l *FollowFeedLogic) buildFollowItems(contents []*model.ZfeedContent) ([]*contentpb.FollowFeedItem, error) {
	articleMap, videoMap, err := l.buildBriefMaps(contents)
	if err != nil {
		return nil, err
	}

	items := make([]*contentpb.FollowFeedItem, 0, len(contents))
	for _, row := range contents {
		if row == nil || row.ID <= 0 {
			continue
		}

		item := &contentpb.FollowFeedItem{
			ContentId:   row.ID,
			ContentType: contentpb.ContentType(row.ContentType),
			AuthorId:    row.UserID,
		}
		if row.PublishedAt != nil {
			item.PublishedAt = row.PublishedAt.Unix()
		}

		switch contentpb.ContentType(row.ContentType) {
		case contentpb.ContentType_CONTENT_TYPE_ARTICLE:
			if article, ok := articleMap[row.ID]; ok && article != nil {
				item.Title = article.Title
				item.CoverUrl = article.Cover
			}
		case contentpb.ContentType_CONTENT_TYPE_VIDEO:
			if video, ok := videoMap[row.ID]; ok && video != nil {
				item.Title = video.Title
				item.CoverUrl = video.CoverURL
			}
		}

		items = append(items, item)
	}
	return items, nil
}

func (l *FollowFeedLogic) buildBriefMaps(contents []*model.ZfeedContent) (map[int64]*model.ZfeedArticle, map[int64]*model.ZfeedVideo, error) {
	articleIDs := make([]int64, 0)
	videoIDs := make([]int64, 0)

	for _, row := range contents {
		if row == nil || row.ID <= 0 {
			continue
		}
		switch contentpb.ContentType(row.ContentType) {
		case contentpb.ContentType_CONTENT_TYPE_ARTICLE:
			articleIDs = append(articleIDs, row.ID)
		case contentpb.ContentType_CONTENT_TYPE_VIDEO:
			videoIDs = append(videoIDs, row.ID)
		}
	}

	articleMap, err := l.articleRepo.BatchGetBriefByContentIDs(articleIDs)
	if err != nil {
		return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询文章摘要失败"))
	}
	videoMap, err := l.videoRepo.BatchGetBriefByContentIDs(videoIDs)
	if err != nil {
		return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询视频摘要失败"))
	}
	return articleMap, videoMap, nil
}
