package interaction

import (
	"context"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logc"

	"zfeed/app/front/internal/svc"
	contentpb "zfeed/app/rpc/content/content"
)

const recommendTrackSourceInteraction = "interaction"

func emitRecommendTrack(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	eventType string,
	userID int64,
	contentID int64,
) {
	eventType = strings.TrimSpace(eventType)
	if svcCtx == nil || svcCtx.FeedRpc == nil || eventType == "" || userID <= 0 || contentID <= 0 {
		return
	}
	if svcCtx.Config.DisableRecommendInteractionTrack {
		return
	}

	_, err := svcCtx.FeedRpc.EmitRecommendTrack(ctx, &contentpb.EmitRecommendTrackReq{
		UserId:     userID,
		EventType:  eventType,
		ContentId:  contentID,
		Source:     recommendTrackSourceInteraction,
		OccurredAt: time.Now().Unix(),
	})
	if err != nil {
		logc.Errorf(
			ctx,
			"emit recommend track from interaction failed, event_type=%s, user_id=%d, content_id=%d, err=%v",
			eventType,
			userID,
			contentID,
			err,
		)
	}
}
