// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type EmitRecommendTrackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEmitRecommendTrackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EmitRecommendTrackLogic {
	return &EmitRecommendTrackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EmitRecommendTrackLogic) EmitRecommendTrack(req *types.RecommendTrackReq) (resp *types.RecommendTrackRes, err error) {
	if req == nil || req.EventType == nil || req.ContentId == nil || *req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	rpcReq := &contentpb.EmitRecommendTrackReq{
		UserId:     utils.GetContextUserIdWithDefault(l.ctx),
		EventType:  *req.EventType,
		ContentId:  *req.ContentId,
		RequestId:  stringOrEmpty(req.RequestId),
		SnapshotId: stringOrEmpty(req.SnapshotId),
		VariantId:  stringOrEmpty(req.VariantId),
		Source:     stringOrEmpty(req.Source),
		Position:   int32(int64OrZero(req.Position)),
		FinalScore: float64OrZero(req.FinalScore),
		DwellMs:    int64OrZero(req.DwellMs),
		OccurredAt: int64OrZero(req.OccurredAt),
	}
	if _, err := l.svcCtx.FeedRpc.EmitRecommendTrack(l.ctx, rpcReq); err != nil {
		return nil, err
	}
	return &types.RecommendTrackRes{}, nil
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int64OrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func float64OrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
