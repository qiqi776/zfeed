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

type CommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	return &CommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommentLogic) Comment(req *types.CommentReq) (resp *types.CommentRes, err error) {
	if req == nil || req.ContentId == nil || req.ContentUserId == nil || req.Scene == nil || req.Comment == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	scene, err := parseScene(*req.Scene)
	if err != nil {
		return nil, err
	}

	res, err := l.svcCtx.CommentRpc.Comment(l.ctx, &commentservicepb.CommentReq{
		UserId:        userID,
		ContentId:     *req.ContentId,
		Scene:         scene,
		Comment:       trimCommentInput(*req.Comment),
		ParentId:      optionalInt64Value(req.ParentId),
		RootId:        optionalInt64Value(req.RootId),
		ReplyToUserId: optionalInt64Value(req.ReplyToUserId),
		ContentUserId: *req.ContentUserId,
	})
	if err != nil {
		return nil, err
	}

	return &types.CommentRes{
		CommentId: res.GetCommentId(),
	}, nil
}
