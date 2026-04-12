package commentservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefillCommentCacheLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewRefillCommentCacheLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefillCommentCacheLogic {
	return &RefillCommentCacheLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RefillCommentCacheLogic) RefillCommentCache(in *interaction.RefillCommentCacheReq) (*interaction.RefillCommentCacheRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	commentIDs := uniqueCommentIDs(in.GetCommentIds())
	if len(commentIDs) == 0 {
		return &interaction.RefillCommentCacheRes{Comments: []*interaction.CommentItem{}}, nil
	}

	items, missIDs, err := loadCommentItemsByIDs(l.ctx, l.commentRepo, l.svcCtx.UserRpc, commentIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("回填评论缓存失败"))
	}
	if len(items) > 0 {
		cacheCommentItemsBestEffort(l.ctx, l.Logger, l.svcCtx.Redis, items)
	}

	return &interaction.RefillCommentCacheRes{
		Comments: filterOrderedCommentItems(commentIDs, items, missIDs),
	}, nil
}
