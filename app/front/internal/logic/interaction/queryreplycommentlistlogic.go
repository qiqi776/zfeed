// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryReplyCommentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryReplyCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryReplyCommentListLogic {
	return &QueryReplyCommentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryReplyCommentListLogic) QueryReplyCommentList(req *types.QueryReplyCommentListReq) (resp *types.QueryReplyCommentListRes, err error) {
	// todo: add your logic here and delete this line

	return
}
