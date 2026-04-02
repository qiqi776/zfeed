package commentservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefillCommentCacheLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefillCommentCacheLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefillCommentCacheLogic {
	return &RefillCommentCacheLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RefillCommentCacheLogic) RefillCommentCache(in *interaction.RefillCommentCacheReq) (*interaction.RefillCommentCacheRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.RefillCommentCacheRes{}, nil
}
