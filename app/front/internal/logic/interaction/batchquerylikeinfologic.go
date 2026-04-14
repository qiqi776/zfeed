// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	interactionpb "zfeed/app/rpc/interaction/interaction"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchQueryLikeInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryLikeInfoLogic {
	return &BatchQueryLikeInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchQueryLikeInfoLogic) BatchQueryLikeInfo(req *types.BatchQueryLikeInfoReq) (resp *types.BatchQueryLikeInfoRes, err error) {
	if req == nil || len(req.LikeInfos) == 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	likeInfos := make([]*interactionpb.LikeInfo, 0, len(req.LikeInfos))
	for _, item := range req.LikeInfos {
		if item.ContentId == nil || item.Scene == nil {
			return nil, errorx.NewBadRequest("参数错误")
		}

		scene, parseErr := parseScene(*item.Scene)
		if parseErr != nil {
			return nil, parseErr
		}

		likeInfos = append(likeInfos, &interactionpb.LikeInfo{
			ContentId: *item.ContentId,
			Scene:     scene,
		})
	}

	userID := utils.GetContextUserIdWithDefault(l.ctx)
	rpcResp, err := l.svcCtx.LikeRpc.BatchQueryLikeInfo(l.ctx, &interactionpb.BatchQueryLikeInfoReq{
		UserId:    userID,
		LikeInfos: likeInfos,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.QueryLikeInfoRes, 0, len(rpcResp.GetLikeInfos()))
	for _, item := range rpcResp.GetLikeInfos() {
		if item == nil {
			continue
		}
		items = append(items, types.QueryLikeInfoRes{
			LikeCount: item.GetLikeCount(),
			IsLiked:   item.GetIsLiked(),
			ContentId: item.GetContentId(),
			Scene:     item.GetScene().String(),
		})
	}

	return &types.BatchQueryLikeInfoRes{
		LikeInfos: items,
	}, nil
}
