package feedservicelogic

import (
	"context"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecommendFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RecommendFeedLogic) RecommendFeed(in *content.RecommendFeedReq) (*content.RecommendFeedRes, error) {
	// todo: add your logic here and delete this line

	return &content.RecommendFeedRes{}, nil
}
