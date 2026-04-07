package strategy

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/changeevent"
)

const favoriteTableName = "zfeed_favorite"

type favoriteStrategy struct{}

func newFavoriteStrategy() TableStrategy {
	return &favoriteStrategy{}
}

func (s *favoriteStrategy) TableName() string {
	return favoriteTableName
}

func (s *favoriteStrategy) ExtractUpdates(ctx context.Context, evt changeevent.ChangeEvent) []Update {
	contentID, ok := getInt64(evt.Current["content_id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "invalid favorite content_id, event=%+v", evt)
		return nil
	}
	ownerID, _ := getInt64(evt.Current["content_user_id"])

	var delta int64
	switch strings.ToUpper(strings.TrimSpace(evt.Operation)) {
	case "INSERT":
		if isFavoriteActive(evt.Current) {
			delta = 1
		}
	case "DELETE":
		if isFavoriteActive(evt.Current) {
			delta = -1
		}
	case "UPDATE":
		before := map[string]any{"status": mergedValue(evt.Current, evt.Previous, "status")}
		delta = boolDelta(isFavoriteActive(before), isFavoriteActive(evt.Current))
	}
	if delta == 0 {
		return nil
	}

	return []Update{{
		BizType:    count.BizType_FAVORITE,
		TargetType: count.TargetType_CONTENT,
		TargetID:   contentID,
		OwnerID:    ownerID,
		Delta:      delta,
	}}
}
