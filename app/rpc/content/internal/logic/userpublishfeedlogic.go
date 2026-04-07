package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
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

func (l *UserPublishFeedLogic) UserPublishFeed(in *contentpb.UserPublishFeedReq) (*contentpb.UserPublishFeedRes, error) {
	if in == nil || in.GetAuthorId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	return &contentpb.UserPublishFeedRes{
		Items:      []*contentpb.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}, nil
}
