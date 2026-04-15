package feedservicelogic

import (
	"context"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFavoriteFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUserFavoriteFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFavoriteFeedLogic {
	return &UserFavoriteFeedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UserFavoriteFeedLogic) UserFavoriteFeed(in *content.UserFavoriteFeedReq) (*content.UserFavoriteFeedRes, error) {
	// todo: add your logic here and delete this line

	return &content.UserFavoriteFeedRes{}, nil
}
