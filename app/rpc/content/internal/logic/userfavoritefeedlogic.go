package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
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

func (l *UserFavoriteFeedLogic) UserFavoriteFeed(in *contentpb.UserFavoriteFeedReq) (*contentpb.UserFavoriteFeedRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	return &contentpb.UserFavoriteFeedRes{
		Items:      []*contentpb.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}, nil
}
