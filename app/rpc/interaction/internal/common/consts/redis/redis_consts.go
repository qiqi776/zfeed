package redis

const (
	LikeUserPrefix           = "like:user"
	LikeExpireSeconds        = 5 * 24 * 60 * 60
	FavoriteRelPrefix        = "favorite:rel"
	FavoriteRelExpireSecs    = 24 * 60 * 60
	FavoriteRelNegExpireSecs = 10 * 60
	UserFavoriteFeedPrefix   = "feed:user:favorite"
	CommentItemPrefix        = "comment:item"
	CommentListPrefix        = "comment:list"
	CommentReplyPrefix       = "comment:reply"
	CommentListLockPrefix    = "lock:comment:list"
	CommentReplyLockPrefix   = "lock:comment:reply"
	CommentItemExpireSecs    = 24 * 60 * 60
	CommentIndexExpireSecs   = 30 * 60
	CommentLockExpireSecs    = 5
)

func BuildLikeUserKey(userID string) string {
	return LikeUserPrefix + ":" + userID
}

func BuildFavoriteRelKey(scene string, userID string, contentID string) string {
	return FavoriteRelPrefix + ":" + scene + ":" + userID + ":" + contentID
}

func BuildUserFavoriteFeedKey(userID string) string {
	return UserFavoriteFeedPrefix + ":" + userID
}

func BuildCommentItemKey(commentID string) string {
	return CommentItemPrefix + ":" + commentID
}

func BuildCommentListKey(scene string, contentID string) string {
	return CommentListPrefix + ":" + scene + ":" + contentID
}

func BuildCommentReplyKey(rootID string) string {
	return CommentReplyPrefix + ":" + rootID
}

func BuildCommentListLockKey(scene string, contentID string) string {
	return CommentListLockPrefix + ":" + scene + ":" + contentID
}

func BuildCommentReplyLockKey(rootID string) string {
	return CommentReplyLockPrefix + ":" + rootID
}
