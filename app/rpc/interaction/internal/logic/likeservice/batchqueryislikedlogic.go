package likeservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchQueryIsLikedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchQueryIsLikedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryIsLikedLogic {
	return &BatchQueryIsLikedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BatchQueryIsLikedLogic) BatchQueryIsLiked(in *interaction.BatchQueryIsLikedReq) (*interaction.BatchQueryIsLikedRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.BatchQueryIsLikedRes{}, nil
}
