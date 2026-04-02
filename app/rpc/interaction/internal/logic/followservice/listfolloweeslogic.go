package followservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFolloweesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFolloweesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFolloweesLogic {
	return &ListFolloweesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListFolloweesLogic) ListFollowees(in *interaction.ListFolloweesReq) (*interaction.ListFolloweesRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.ListFolloweesRes{}, nil
}
