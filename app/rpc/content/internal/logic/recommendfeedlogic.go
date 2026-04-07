package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
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

func (l *RecommendFeedLogic) RecommendFeed(*contentpb.RecommendFeedReq) (*contentpb.RecommendFeedRes, error) {
	return &contentpb.RecommendFeedRes{
		Items:      []*contentpb.ContentItem{},
		NextCursor: 0,
		HasMore:    false,
		SnapshotId: "",
	}, nil
}
