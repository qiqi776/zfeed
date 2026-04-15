package content

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContentLogic {
	return &DeleteContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteContentLogic) DeleteContent(req *types.DeleteContentReq) (*types.DeleteContentRes, error) {
	if req == nil || req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	if _, err := l.svcCtx.ContentRpc.DeleteContent(l.ctx, &contentservice.DeleteContentReq{
		UserId:    userID,
		ContentId: req.ContentId,
	}); err != nil {
		return nil, err
	}

	return &types.DeleteContentRes{}, nil
}
