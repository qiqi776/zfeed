package commentservicelogic

import (
	"context"
	"errors"
	"strconv"
	"time"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryCommentListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewQueryCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryCommentListLogic {
	return &QueryCommentListLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryCommentListLogic) QueryCommentList(in *interaction.QueryCommentListReq) (*interaction.QueryCommentListRes, error) {
	if in == nil || in.GetContentId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetScene() == interaction.Scene_SCENE_UNKNOWN {
		return nil, errorx.NewMsg("场景参数错误")
	}

	pageSize, err := normalizeCommentPage(in.GetPageSize())
	if err != nil {
		return nil, err
	}

	if res, ok := l.queryFromCache(in, pageSize); ok {
		return res, nil
	}

	return l.queryWithRebuild(in, pageSize)
}

func (l *QueryCommentListLogic) queryWithRebuild(in *interaction.QueryCommentListReq, pageSize uint32) (*interaction.QueryCommentListRes, error) {
	lockKey := rediskey.BuildCommentListLockKey(in.GetScene().String(), strconv.FormatInt(in.GetContentId(), 10))
	locked, err := tryAcquireCommentRebuildLock(l.ctx, l.svcCtx.Redis, lockKey)
	if err != nil {
		l.Errorf("获取评论列表重建锁失败: %v, content_id=%d", err, in.GetContentId())
	}
	if locked {
		defer releaseCommentRebuildLock(l.ctx, l.Logger, l.svcCtx.Redis, lockKey)
		return l.queryFromDB(in, pageSize, true)
	}

	for i := 0; i < commentRebuildRetryTimes; i++ {
		time.Sleep(commentRebuildRetryDelay)
		if res, ok := l.queryFromCache(in, pageSize); ok {
			return res, nil
		}
	}

	return l.queryFromDB(in, pageSize, false)
}

func (l *QueryCommentListLogic) queryFromDB(in *interaction.QueryCommentListReq, pageSize uint32, rebuildCache bool) (*interaction.QueryCommentListRes, error) {

	comments, err := l.commentRepo.ListRootCommentsIncludeDeleted(in.GetContentId(), in.GetCursor(), pageSize)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论列表失败"))
	}

	trimmed, nextCursor, hasMore := trimCommentPage(comments, pageSize)
	userMap, err := batchLoadCommentUsers(l.ctx, l.svcCtx.UserRpc, trimmed)
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论用户失败"))
	}

	res := &interaction.QueryCommentListRes{
		Comments:   buildCommentItems(trimmed, userMap),
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
	if rebuildCache {
		l.cacheDBResultBestEffort(in.GetScene(), in.GetContentId(), res.GetComments())
	}

	return res, nil
}

func (l *QueryCommentListLogic) queryFromCache(in *interaction.QueryCommentListReq, pageSize uint32) (*interaction.QueryCommentListRes, bool) {
	key := rediskey.BuildCommentListKey(in.GetScene().String(), strconv.FormatInt(in.GetContentId(), 10))
	ids, exists, err := readCachedCommentIndexIDs(l.ctx, l.svcCtx.Redis, key, in.GetCursor(), pageSize)
	if err != nil {
		l.Errorf("读取评论列表缓存失败: %v, content_id=%d", err, in.GetContentId())
		return nil, false
	}
	if !exists {
		return nil, false
	}
	if len(ids) == 0 {
		return &interaction.QueryCommentListRes{
			Comments:   []*interaction.CommentItem{},
			NextCursor: 0,
			HasMore:    false,
		}, true
	}

	cachedMap, missIDs, err := readCachedCommentItems(l.ctx, l.svcCtx.Redis, ids)
	if err != nil {
		l.Errorf("读取评论对象缓存失败: %v, content_id=%d", err, in.GetContentId())
		return nil, false
	}
	if len(missIDs) > 0 {
		refillRes, refillErr := NewRefillCommentCacheLogic(l.ctx, l.svcCtx).RefillCommentCache(&interaction.RefillCommentCacheReq{
			CommentIds: missIDs,
		})
		if refillErr != nil {
			l.Errorf("回填评论对象缓存失败: %v, content_id=%d", refillErr, in.GetContentId())
			return nil, false
		}
		mergeCommentItems(refillRes.GetComments(), cachedMap)
		for _, commentID := range missIDs {
			if cachedMap[commentID] == nil {
				invalidateCommentCacheKeysBestEffort(l.ctx, l.Logger, l.svcCtx.Redis, key)
				return nil, false
			}
		}
	}

	items, nextCursor, hasMore, ok := buildCachedCommentResult(ids, cachedMap, pageSize)
	if !ok {
		return nil, false
	}

	return &interaction.QueryCommentListRes{
		Comments:   items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, true
}

func (l *QueryCommentListLogic) cacheDBResultBestEffort(scene interaction.Scene, contentID int64, items []*interaction.CommentItem) {
	if len(items) == 0 {
		return
	}
	cacheCommentItemsAndIndexBestEffort(
		l.ctx,
		l.Logger,
		l.svcCtx.Redis,
		rediskey.BuildCommentListKey(scene.String(), strconv.FormatInt(contentID, 10)),
		items,
	)
}
