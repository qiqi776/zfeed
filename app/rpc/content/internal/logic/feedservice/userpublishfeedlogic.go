package feedservicelogic

import (
	"context"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserPublishFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUserPublishFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishFeedLogic {
	return &UserPublishFeedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UserPublishFeedLogic) UserPublishFeed(in *content.UserPublishFeedReq) (*content.UserPublishFeedRes, error) {
	// todo: add your logic here and delete this line

	return &content.UserPublishFeedRes{}, nil
}
