// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/interaction/interaction"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type LikeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LikeLogic) Like(req *types.LikeReq) (resp *types.LikeRes, err error) {
	if req == nil || req.ContentId == nil || req.ContentUserId == nil || req.Scene == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	scene, err := parseScene(*req.Scene)
	if err != nil {
		return nil, err
	}

	_, err = l.svcCtx.LikeRpc.Like(l.ctx, &interaction.LikeReq{
		UserId:        userID,
		ContentId:     *req.ContentId,
		ContentUserId: *req.ContentUserId,
		Scene:         scene,
	})
	if err != nil {
		return nil, err
	}
	return &types.LikeRes{}, nil
}
