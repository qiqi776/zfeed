package likeservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchQueryLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewBatchQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryLikeInfoLogic {
	return &BatchQueryLikeInfoLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchQueryLikeInfoLogic) BatchQueryLikeInfo(in *interaction.BatchQueryLikeInfoReq) (*interaction.BatchQueryLikeInfoRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	normalized := normalizeLikeInfos(in.GetLikeInfos())
	if len(normalized) == 0 {
		return &interaction.BatchQueryLikeInfoRes{
			LikeInfos: []*interaction.QueryLikeInfoRes{},
		}, nil
	}

	contentIDs := make([]int64, 0, len(normalized))
	for _, item := range normalized {
		contentIDs = append(contentIDs, item.contentID)
	}

	countMap, err := l.likeRepo.CountByContentIDs(contentIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
	}

	isLikedMap := map[int64]bool{}
	if in.GetUserId() > 0 {
		stateLoader := NewBatchQueryIsLikedLogic(l.ctx, l.svcCtx)
		isLikedMap, err = stateLoader.loadBatchLikedState(in.GetUserId(), contentIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
		}
	}

	items := make([]*interaction.QueryLikeInfoRes, 0, len(normalized))
	for _, item := range normalized {
		items = append(items, &interaction.QueryLikeInfoRes{
			LikeCount: countMap[item.contentID],
			IsLiked:   isLikedMap[item.contentID],
			ContentId: item.contentID,
			Scene:     item.scene,
		})
	}

	return &interaction.BatchQueryLikeInfoRes{
		LikeInfos: items,
	}, nil
}
