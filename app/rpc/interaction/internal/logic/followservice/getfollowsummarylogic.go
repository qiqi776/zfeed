package followservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFollowSummaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFollowSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFollowSummaryLogic {
	return &GetFollowSummaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetFollowSummaryLogic) GetFollowSummary(in *interaction.GetFollowSummaryReq) (*interaction.GetFollowSummaryRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.GetFollowSummaryRes{}, nil
}
