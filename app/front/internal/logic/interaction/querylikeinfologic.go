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

type QueryLikeInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLikeInfoLogic {
	return &QueryLikeInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryLikeInfoLogic) QueryLikeInfo(req *types.QueryLikeInfoReq) (resp *types.QueryLikeInfoRes, err error) {
	if req == nil || req.ContentId == nil || req.Scene == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	scene, err := parseScene(*req.Scene)
	if err != nil {
		return nil, err
	}

	userID := utils.GetContextUserIdWithDefault(l.ctx)
	rpcResp, err := l.svcCtx.LikeRpc.QueryLikeInfo(l.ctx, &interactionpb.QueryLikeInfoReq{
		UserId:    userID,
		ContentId: *req.ContentId,
		Scene:     scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryLikeInfoRes{
		LikeCount: rpcResp.GetLikeCount(),
		IsLiked:   rpcResp.GetIsLiked(),
		ContentId: rpcResp.GetContentId(),
		Scene:     rpcResp.GetScene().String(),
	}, nil
}
