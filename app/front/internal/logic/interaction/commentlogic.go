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
	if req == nil || req.ContentId == nil || *req.ContentId <= 0 ||
		req.ContentUserId == nil || *req.ContentUserId <= 0 || req.Scene == nil || req.Comment == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if negativeOptionalInt64(req.ParentId) || negativeOptionalInt64(req.RootId) || negativeOptionalInt64(req.ReplyToUserId) {
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

	comment := trimCommentInput(*req.Comment)
	if comment == "" {
		return nil, errorx.NewBadRequest("评论内容不能为空")
	}

	res, err := l.svcCtx.CommentRpc.Comment(l.ctx, &commentservicepb.CommentReq{
		UserId:        userID,
		ContentId:     *req.ContentId,
		Scene:         scene,
		Comment:       comment,
		ParentId:      optionalInt64Value(req.ParentId),
		RootId:        optionalInt64Value(req.RootId),
		ReplyToUserId: optionalInt64Value(req.ReplyToUserId),
		ContentUserId: *req.ContentUserId,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errorx.NewMsg("发表评论失败")
	}

	return &types.CommentRes{
		CommentId: res.GetCommentId(),
	}, nil
}
