package commentservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryReplyListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryReplyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryReplyListLogic {
	return &QueryReplyListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *QueryReplyListLogic) QueryReplyList(in *interaction.QueryReplyListReq) (*interaction.QueryReplyListRes, error) {
	// todo: add your logic here and delete this line

	return &interaction.QueryReplyListRes{}, nil
}
