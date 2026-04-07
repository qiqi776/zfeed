package strategy

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/changeevent"
)

const followTableName = "zfeed_follow"

type followStrategy struct{}

func newFollowStrategy() TableStrategy {
	return &followStrategy{}
}

func (s *followStrategy) TableName() string {
	return followTableName
}

func (s *followStrategy) ExtractUpdates(ctx context.Context, evt changeevent.ChangeEvent) []Update {
	userID, ok := getInt64(evt.Current["user_id"])
	if !ok || userID <= 0 {
		logc.Errorf(ctx, "invalid follow user_id, event=%+v", evt)
		return nil
	}
	followUserID, ok := getInt64(evt.Current["follow_user_id"])
	if !ok || followUserID <= 0 {
		logc.Errorf(ctx, "invalid follow follow_user_id, event=%+v", evt)
		return nil
	}

	var delta int64
	switch strings.ToUpper(strings.TrimSpace(evt.Operation)) {
	case "INSERT":
		if isFollowActive(evt.Current) {
			delta = 1
		}
	case "DELETE":
		if isFollowActive(evt.Current) {
			delta = -1
		}
	case "UPDATE":
		before := map[string]any{
			"status":     mergedValue(evt.Current, evt.Previous, "status"),
			"is_deleted": mergedValue(evt.Current, evt.Previous, "is_deleted"),
		}
		delta = boolDelta(isFollowActive(before), isFollowActive(evt.Current))
	}
	if delta == 0 {
		return nil
	}

	return []Update{
		{
			BizType:    count.BizType_FOLLOWING,
			TargetType: count.TargetType_USER,
			TargetID:   userID,
			Delta:      delta,
		},
		{
			BizType:    count.BizType_FOLLOWED,
			TargetType: count.TargetType_USER,
			TargetID:   followUserID,
			Delta:      delta,
		},
	}
}
