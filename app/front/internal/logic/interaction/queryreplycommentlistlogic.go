// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	commentservicepb "zfeed/app/rpc/interaction/client/commentservice"
	"zfeed/pkg/errorx"

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
	if req == nil || req.CommentId == nil || req.Cursor == nil || req.PageSize == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	res, err := l.svcCtx.CommentRpc.QueryReplyList(l.ctx, &commentservicepb.QueryReplyListReq{
		RootId:   *req.CommentId,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryReplyCommentListRes{
		Comments:   commentItemsFromRPC(res.GetReplies()),
		NextCursor: res.GetNextCursor(),
		HasMore:    res.GetHasMore(),
	}, nil
}
