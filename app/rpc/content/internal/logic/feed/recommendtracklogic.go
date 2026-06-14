package feedlogic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
)

type RecommendTrackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecommendTrackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendTrackLogic {
	return &RecommendTrackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RecommendTrackLogic) EmitRecommendTrack(
	in *contentpb.EmitRecommendTrackReq,
) (*contentpb.EmitRecommendTrackRes, error) {
	event, err := l.buildTrackEvent(in)
	if err != nil {
		return nil, err
	}

	if l.svcCtx == nil || l.svcCtx.RecommendTrackProducer == nil {
		recordRecommendTrackEmitMetric(event.EventType, recommendResultError)
		return nil, errorx.NewMsg("推荐埋点生产者未初始化")
	}
	if err := l.svcCtx.RecommendTrackProducer.Emit(l.ctx, event); err != nil {
		recordRecommendTrackEmitMetric(event.EventType, recommendResultError)
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("上报推荐埋点失败"))
	}

	recordRecommendTrackEmitMetric(event.EventType, recommendResultSuccess)
	return &contentpb.EmitRecommendTrackRes{}, nil
}

func (l *RecommendTrackLogic) buildTrackEvent(in *contentpb.EmitRecommendTrackReq) (track.Event, error) {
	if in == nil || in.GetContentId() <= 0 {
		return track.Event{}, errorx.NewBadRequest("参数错误")
	}

	eventType := strings.TrimSpace(in.GetEventType())
	if !track.IsClientEventType(eventType) {
		return track.Event{}, errorx.NewBadRequest("埋点事件类型错误")
	}
	if eventType == track.EventTypeDwell && in.GetDwellMs() <= 0 {
		return track.Event{}, errorx.NewBadRequest("停留时长参数错误")
	}

	occurredAt := in.GetOccurredAt()
	if occurredAt <= 0 {
		occurredAt = time.Now().Unix()
	}

	return track.Event{
		EventID:    buildClientTrackEventID(in.GetUserId(), eventType, in.GetContentId(), occurredAt),
		EventType:  eventType,
		UserID:     in.GetUserId(),
		ContentID:  in.GetContentId(),
		RequestID:  strings.TrimSpace(in.GetRequestId()),
		SnapshotID: strings.TrimSpace(in.GetSnapshotId()),
		VariantID:  strings.TrimSpace(in.GetVariantId()),
		Source:     strings.TrimSpace(in.GetSource()),
		Position:   int(in.GetPosition()),
		FinalScore: in.GetFinalScore(),
		DwellMs:    in.GetDwellMs(),
		OccurredAt: occurredAt,
	}, nil
}

func buildClientTrackEventID(userID int64, eventType string, contentID, occurredAt int64) string {
	return fmt.Sprintf(
		"rec_%s_%d_%d_%d_%d",
		eventType,
		userID,
		contentID,
		occurredAt,
		time.Now().UnixNano(),
	)
}
