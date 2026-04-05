package interaction

import (
	"strings"

	"zfeed/app/front/internal/types"
	commentservicepb "zfeed/app/rpc/interaction/client/commentservice"
)

func optionalInt64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func commentItemsFromRPC(items []*commentservicepb.CommentItem) []*types.CommentItem {
	result := make([]*types.CommentItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, &types.CommentItem{
			CommentId:     item.GetCommentId(),
			ContentId:     item.GetContentId(),
			UserId:        item.GetUserId(),
			ReplyToUserId: item.GetReplyToUserId(),
			ParentId:      item.GetParentId(),
			RootId:        item.GetRootId(),
			Comment:       item.GetComment(),
			CreatedAt:     item.GetCreatedAt(),
			Status:        item.GetStatus(),
			UserName:      item.GetUserName(),
			UserAvatar:    item.GetUserAvatar(),
			ReplyCount:    item.GetReplyCount(),
		})
	}
	return result
}

func trimCommentInput(raw string) string {
	return strings.TrimSpace(raw)
}
