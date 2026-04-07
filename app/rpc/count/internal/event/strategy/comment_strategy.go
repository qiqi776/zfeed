package strategy

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/changeevent"
)

const commentTableName = "zfeed_comment"

type commentStrategy struct{}

func newCommentStrategy() TableStrategy {
	return &commentStrategy{}
}

func (s *commentStrategy) TableName() string {
	return commentTableName
}

func (s *commentStrategy) ExtractUpdates(ctx context.Context, evt changeevent.ChangeEvent) []Update {
	contentID, ok := getInt64(evt.Current["content_id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "invalid comment content_id, event=%+v", evt)
		return nil
	}
	ownerID, _ := getInt64(evt.Current["content_user_id"])

	var delta int64
	switch strings.ToUpper(strings.TrimSpace(evt.Operation)) {
	case "INSERT":
		if isCommentActive(evt.Current) {
			delta = 1
		}
	case "DELETE":
		before := map[string]any{
			"status":     mergedValue(evt.Current, evt.Previous, "status"),
			"is_deleted": mergedValue(evt.Current, evt.Previous, "is_deleted"),
		}
		if isCommentActive(before) {
			delta = -1
		}
	case "UPDATE":
		before := map[string]any{
			"status":     mergedValue(evt.Current, evt.Previous, "status"),
			"is_deleted": mergedValue(evt.Current, evt.Previous, "is_deleted"),
		}
		delta = boolDelta(isCommentActive(before), isCommentActive(evt.Current))
	}
	if delta == 0 {
		return nil
	}

	return []Update{{
		BizType:    count.BizType_COMMENT,
		TargetType: count.TargetType_CONTENT,
		TargetID:   contentID,
		OwnerID:    ownerID,
		Delta:      delta,
	}}
}
