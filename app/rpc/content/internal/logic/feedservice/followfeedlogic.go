package feedservicelogic

import (
	"context"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
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

func (l *FollowFeedLogic) FollowFeed(in *content.FollowFeedReq) (*content.FollowFeedRes, error) {
	// todo: add your logic here and delete this line

	return &content.FollowFeedRes{}, nil
}
