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

type QueryCommentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryCommentListLogic {
	return &QueryCommentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryCommentListLogic) QueryCommentList(req *types.QueryCommentListReq) (resp *types.QueryCommentListRes, err error) {
	if req == nil || req.ContentId == nil || req.Scene == nil || req.Cursor == nil || req.PageSize == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	scene, err := parseScene(*req.Scene)
	if err != nil {
		return nil, err
	}

	res, err := l.svcCtx.CommentRpc.QueryCommentList(l.ctx, &commentservicepb.QueryCommentListReq{
		ContentId: *req.ContentId,
		Scene:     scene,
		Cursor:    *req.Cursor,
		PageSize:  *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryCommentListRes{
		Comments:   commentItemsFromRPC(res.GetComments()),
		NextCursor: res.GetNextCursor(),
		HasMore:    res.GetHasMore(),
	}, nil
}
