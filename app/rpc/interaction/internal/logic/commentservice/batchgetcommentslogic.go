package commentservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetCommentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetCommentsLogic {
	return &BatchGetCommentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BatchGetCommentsLogic) BatchGetComments(in *interaction.BatchGetCommentsReq) (*interaction.BatchGetCommentsRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.BatchGetCommentsRes{}, nil
}
