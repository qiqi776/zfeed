// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"
)

type PublishVideoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishVideoLogic {
	return &PublishVideoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishVideoLogic) PublishVideo(req *types.PublishVideoReq) (resp *types.PublishVideoRes, err error) {
	if req == nil || req.Title == nil || req.VideoUrl == nil || req.CoverUrl == nil || req.Visibility == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	var duration int32
	if req.Duration != nil {
		duration = *req.Duration
	}

	rpcResp, err := l.svcCtx.ContentRpc.PublishVideo(l.ctx, &contentpb.VideoPublishReq{
		UserId:      userID,
		Title:       strings.TrimSpace(*req.Title),
		Description: req.Description,
		OriginUrl:   strings.TrimSpace(*req.VideoUrl),
		CoverUrl:    strings.TrimSpace(*req.CoverUrl),
		Duration:    duration,
		Visibility:  contentpb.Visibility(*req.Visibility),
	})
	if err != nil {
		return nil, err
	}

	return &types.PublishVideoRes{
		ContentId: rpcResp.GetContentId(),
	}, nil
}
