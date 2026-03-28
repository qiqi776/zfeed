// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserPublishLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishLogic {
	return &UserPublishLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserPublishLogic) UserPublish(req *types.UserPublishFeedReq) (resp *types.UserPublishFeedRes, err error) {
	// todo: add your logic here and delete this line

	return
}
