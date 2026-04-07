package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
)

type FollowFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFollowFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowFeedLogic {
	return &FollowFeedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FollowFeedLogic) FollowFeed(in *contentpb.FollowFeedReq) (*contentpb.FollowFeedRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	return &contentpb.FollowFeedRes{
		Items:      []*contentpb.FollowFeedItem{},
		NextCursor: "",
		HasMore:    false,
	}, nil
}
