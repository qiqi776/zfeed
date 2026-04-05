package commentservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"
)

const maxCommentPageSize uint32 = 50

func normalizeCommentPage(pageSize uint32) (uint32, error) {
	if pageSize == 0 || pageSize > maxCommentPageSize {
		return 0, errorx.NewMsg("分页参数错误")
	}
	return pageSize, nil
}

func trimCommentPage(comments []*do.CommentDO, pageSize uint32) ([]*do.CommentDO, int64, bool) {
	if uint32(len(comments)) <= pageSize {
		return comments, 0, false
	}

	trimmed := comments[:pageSize]
	return trimmed, trimmed[len(trimmed)-1].ID, true
}

func collectCommentUserIDs(comments []*do.CommentDO) []int64 {
	seen := make(map[int64]struct{}, len(comments))
	userIDs := make([]int64, 0, len(comments))
	for _, comment := range comments {
		if comment == nil || comment.UserID <= 0 || isDeletedComment(comment) {
			continue
		}
		if _, ok := seen[comment.UserID]; ok {
			continue
		}
		seen[comment.UserID] = struct{}{}
		userIDs = append(userIDs, comment.UserID)
	}
	return userIDs
}

func batchLoadCommentUsers(ctx context.Context, userRPC userservice.UserService, comments []*do.CommentDO) (map[int64]*userservice.UserInfo, error) {
	userIDs := collectCommentUserIDs(comments)
	if len(userIDs) == 0 || userRPC == nil {
		return map[int64]*userservice.UserInfo{}, nil
	}

	res, err := userRPC.BatchGetUser(ctx, &userservice.BatchGetUserReq{UserIds: userIDs})
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]*userservice.UserInfo, len(res.GetUsers()))
	for _, info := range res.GetUsers() {
		if info == nil || info.GetUserId() <= 0 {
			continue
		}
		userMap[info.GetUserId()] = info
	}
	return userMap, nil
}

func isDeletedComment(comment *do.CommentDO) bool {
	if comment == nil {
		return false
	}
	return comment.IsDeleted == 1 || comment.Status == repositories.CommentStatusDeleted
}

func buildCommentItems(comments []*do.CommentDO, userMap map[int64]*userservice.UserInfo) []*interaction.CommentItem {
	items := make([]*interaction.CommentItem, 0, len(comments))
	for _, comment := range comments {
		if comment == nil {
			continue
		}

		item := &interaction.CommentItem{
			CommentId:     comment.ID,
			ContentId:     comment.ContentID,
			UserId:        comment.UserID,
			ReplyToUserId: comment.ReplyToUserID,
			ParentId:      comment.ParentID,
			RootId:        comment.RootID,
			Comment:       comment.Comment,
			CreatedAt:     comment.CreatedAt.Unix(),
			Status:        comment.Status,
			ReplyCount:    comment.ReplyCount,
		}

		if isDeletedComment(comment) {
			item.UserId = 0
			item.Comment = "该评论已删除"
			item.Status = repositories.CommentStatusDeleted
		} else if userInfo := userMap[comment.UserID]; userInfo != nil {
			item.UserName = userInfo.GetNickname()
			item.UserAvatar = userInfo.GetAvatar()
		}

		items = append(items, item)
	}
	return items
}

func loadCommentItemsByIDs(ctx context.Context, commentRepo repositories.CommentRepository, userRPC userservice.UserService, commentIDs []int64) ([]*interaction.CommentItem, []int64, error) {
	ids := uniqueCommentIDs(commentIDs)
	if len(ids) == 0 {
		return []*interaction.CommentItem{}, []int64{}, nil
	}

	commentMap, err := commentRepo.BatchGetByIDsIncludeDeleted(ids)
	if err != nil {
		return nil, nil, err
	}

	orderedComments := make([]*do.CommentDO, 0, len(ids))
	missIDs := make([]int64, 0)
	for _, commentID := range ids {
		commentDO := commentMap[commentID]
		if commentDO == nil {
			missIDs = append(missIDs, commentID)
			continue
		}
		orderedComments = append(orderedComments, commentDO)
	}

	userMap, err := batchLoadCommentUsers(ctx, userRPC, orderedComments)
	if err != nil {
		return nil, nil, err
	}

	return buildCommentItems(orderedComments, userMap), missIDs, nil
}

func mergeCommentItems(items []*interaction.CommentItem, itemMap map[int64]*interaction.CommentItem) {
	for _, item := range items {
		if item == nil || item.GetCommentId() <= 0 {
			continue
		}
		itemMap[item.GetCommentId()] = item
	}
}

func filterOrderedCommentItems(commentIDs []int64, items []*interaction.CommentItem, missIDs []int64) []*interaction.CommentItem {
	if len(items) == 0 {
		return []*interaction.CommentItem{}
	}

	itemMap := make(map[int64]*interaction.CommentItem, len(items))
	mergeCommentItems(items, itemMap)

	missSet := make(map[int64]struct{}, len(missIDs))
	for _, commentID := range missIDs {
		missSet[commentID] = struct{}{}
	}

	result := make([]*interaction.CommentItem, 0, len(items))
	for _, commentID := range uniqueCommentIDs(commentIDs) {
		if _, ok := missSet[commentID]; ok {
			continue
		}
		item := itemMap[commentID]
		if item == nil {
			continue
		}
		result = append(result, item)
	}

	return result
}

func uniqueCommentIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
