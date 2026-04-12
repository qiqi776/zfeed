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

type QueryReplyListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewQueryReplyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryReplyListLogic {
	return &QueryReplyListLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryReplyListLogic) QueryReplyList(in *interaction.QueryReplyListReq) (*interaction.QueryReplyListRes, error) {
	if in == nil || in.GetRootId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
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

func (l *QueryReplyListLogic) queryWithRebuild(in *interaction.QueryReplyListReq, pageSize uint32) (*interaction.QueryReplyListRes, error) {
	lockKey := rediskey.BuildCommentReplyLockKey(strconv.FormatInt(in.GetRootId(), 10))
	locked, err := tryAcquireCommentRebuildLock(l.ctx, l.svcCtx.Redis, lockKey)
	if err != nil {
		l.Errorf("获取回复列表重建锁失败: %v, root_id=%d", err, in.GetRootId())
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

func (l *QueryReplyListLogic) queryFromDB(in *interaction.QueryReplyListReq, pageSize uint32, rebuildCache bool) (*interaction.QueryReplyListRes, error) {

	replies, err := l.commentRepo.ListRepliesIncludeDeleted(in.GetRootId(), in.GetCursor(), pageSize)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询回复列表失败"))
	}

	trimmed, nextCursor, hasMore := trimCommentPage(replies, pageSize)
	userMap, err := batchLoadCommentUsers(l.ctx, l.svcCtx.UserRpc, trimmed)
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询回复用户失败"))
	}

	res := &interaction.QueryReplyListRes{
		RootId:     in.GetRootId(),
		Replies:    buildCommentItems(trimmed, userMap),
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
	if rebuildCache {
		l.cacheDBResultBestEffort(in.GetRootId(), res.GetReplies())
	}

	return res, nil
}

func (l *QueryReplyListLogic) queryFromCache(in *interaction.QueryReplyListReq, pageSize uint32) (*interaction.QueryReplyListRes, bool) {
	key := rediskey.BuildCommentReplyKey(strconv.FormatInt(in.GetRootId(), 10))
	ids, exists, err := readCachedCommentIndexIDs(l.ctx, l.svcCtx.Redis, key, in.GetCursor(), pageSize)
	if err != nil {
		l.Errorf("读取回复列表缓存失败: %v, root_id=%d", err, in.GetRootId())
		return nil, false
	}
	if !exists {
		return nil, false
	}
	if len(ids) == 0 {
		return &interaction.QueryReplyListRes{
			RootId:     in.GetRootId(),
			Replies:    []*interaction.CommentItem{},
			NextCursor: 0,
			HasMore:    false,
		}, true
	}

	cachedMap, missIDs, err := readCachedCommentItems(l.ctx, l.svcCtx.Redis, ids)
	if err != nil {
		l.Errorf("读取回复对象缓存失败: %v, root_id=%d", err, in.GetRootId())
		return nil, false
	}
	if len(missIDs) > 0 {
		refillRes, refillErr := NewRefillCommentCacheLogic(l.ctx, l.svcCtx).RefillCommentCache(&interaction.RefillCommentCacheReq{
			CommentIds: missIDs,
		})
		if refillErr != nil {
			l.Errorf("回填回复对象缓存失败: %v, root_id=%d", refillErr, in.GetRootId())
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

	return &interaction.QueryReplyListRes{
		RootId:     in.GetRootId(),
		Replies:    items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, true
}

func (l *QueryReplyListLogic) cacheDBResultBestEffort(rootID int64, items []*interaction.CommentItem) {
	if len(items) == 0 {
		return
	}
	cacheCommentItemsAndIndexBestEffort(
		l.ctx,
		l.Logger,
		l.svcCtx.Redis,
		rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10)),
		items,
	)
}
