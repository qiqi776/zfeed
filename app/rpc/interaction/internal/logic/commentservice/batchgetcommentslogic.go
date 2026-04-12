package commentservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetCommentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewBatchGetCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetCommentsLogic {
	return &BatchGetCommentsLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchGetCommentsLogic) BatchGetComments(in *interaction.BatchGetCommentsReq) (*interaction.BatchGetCommentsRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}
	commentIDs := uniqueCommentIDs(in.GetCommentIds())
	if len(commentIDs) == 0 {
		return &interaction.BatchGetCommentsRes{
			Comments: []*interaction.CommentItem{},
			MissIds:  []int64{},
		}, nil
	}

	cachedMap, missIDs, err := readCachedCommentItems(l.ctx, l.svcCtx.Redis, commentIDs)
	if err != nil {
		l.Errorf("批量读取评论缓存失败: %v", err)
		cachedMap = map[int64]*interaction.CommentItem{}
		missIDs = commentIDs
	}

	if len(missIDs) > 0 {
		dbItems, stillMiss, dbErr := loadCommentItemsByIDs(l.ctx, l.commentRepo, l.svcCtx.UserRpc, missIDs)
		if dbErr != nil {
			return nil, errorx.Wrap(l.ctx, dbErr, errorx.NewMsg("批量查询评论失败"))
		}

		cacheCommentItemsBestEffort(l.ctx, l.Logger, l.svcCtx.Redis, dbItems)
		mergeCommentItems(dbItems, cachedMap)
		missIDs = stillMiss
	}

	orderedItems := make([]*interaction.CommentItem, 0, len(commentIDs))
	for _, commentID := range commentIDs {
		item := cachedMap[commentID]
		if item == nil {
			continue
		}
		orderedItems = append(orderedItems, item)
	}

	return &interaction.BatchGetCommentsRes{
		Comments: orderedItems,
		MissIds:  missIDs,
	}, nil
}
