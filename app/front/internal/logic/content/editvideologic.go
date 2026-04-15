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

type EditVideoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEditVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditVideoLogic {
	return &EditVideoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditVideoLogic) EditVideo(req *types.EditVideoReq) (*types.EditVideoRes, error) {
	if req == nil || req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewUnauthorized("用户未登录"))
	}

	rpcResp, err := l.svcCtx.ContentRpc.EditVideo(l.ctx, &contentservice.EditVideoReq{
		UserId:      userID,
		ContentId:   req.ContentId,
		Title:       req.Title,
		Description: req.Description,
		OriginUrl:   req.VideoUrl,
		CoverUrl:    req.CoverUrl,
		Duration:    req.Duration,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("更新视频失败")
	}

	return &types.EditVideoRes{ContentId: rpcResp.GetContentId()}, nil
}
