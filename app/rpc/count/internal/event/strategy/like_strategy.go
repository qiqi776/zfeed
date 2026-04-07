package strategy

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/changeevent"
)

const likeTableName = "zfeed_like"

type likeStrategy struct{}

func newLikeStrategy() TableStrategy {
	return &likeStrategy{}
}

func (s *likeStrategy) TableName() string {
	return likeTableName
}

func (s *likeStrategy) ExtractUpdates(ctx context.Context, evt changeevent.ChangeEvent) []Update {
	contentID, ok := getInt64(evt.Current["content_id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "invalid like content_id, event=%+v", evt)
		return nil
	}
	ownerID, _ := getInt64(evt.Current["content_user_id"])

	var delta int64
	switch strings.ToUpper(strings.TrimSpace(evt.Operation)) {
	case "INSERT":
		if isLikeActive(evt.Current) {
			delta = 1
		}
	case "DELETE":
		if isLikeActive(evt.Current) {
			delta = -1
		}
	case "UPDATE":
		before := map[string]any{"status": mergedValue(evt.Current, evt.Previous, "status")}
		delta = boolDelta(isLikeActive(before), isLikeActive(evt.Current))
	}
	if delta == 0 {
		return nil
	}

	return []Update{{
		BizType:    count.BizType_LIKE,
		TargetType: count.TargetType_CONTENT,
		TargetID:   contentID,
		OwnerID:    ownerID,
		Delta:      delta,
	}}
}
