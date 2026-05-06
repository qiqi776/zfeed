package likelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewBatchLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchLikeInfoLogic {
	return &BatchLikeInfoLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchLikeInfoLogic) BatchLikeInfo(in *interaction.BatchLikeInfoReq) (*interaction.BatchLikeInfoRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	normalized := normalizeInfos(in.GetLikeInfos())
	if len(normalized) == 0 {
		return &interaction.BatchLikeInfoRes{
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
		stateLoader := NewBatchIsLikedLogic(l.ctx, l.svcCtx)
		isLikedMap, err = stateLoader.loadLikedMap(in.GetUserId(), contentIDs)
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

	return &interaction.BatchLikeInfoRes{
		LikeInfos: items,
	}, nil
}
