// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	commentservicepb "zfeed/app/rpc/interaction/client/commentservice"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCommentLogic {
	return &DeleteCommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteCommentLogic) DeleteComment(req *types.DeleteCommentReq) (resp *types.DeleteCommentRes, err error) {
	if req == nil || req.CommentId == nil || req.ContentId == nil || req.Scene == nil {
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

	_, err = l.svcCtx.CommentRpc.DeleteComment(l.ctx, &commentservicepb.DeleteCommentReq{
		UserId:    userID,
		CommentId: *req.CommentId,
		ContentId: *req.ContentId,
		RootId:    req.RootId,
		ParentId:  req.ParentId,
		Scene:     scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.DeleteCommentRes{}, nil
}
