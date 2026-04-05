package redis

const (
	RedisLikeUserPrefix         = "like:user"
	RedisLikeExpireSeconds      = 5 * 24 * 60 * 60
	RedisCommentItemPrefix      = "comment:item"
	RedisCommentListPrefix      = "comment:list"
	RedisCommentReplyPrefix     = "comment:reply"
	RedisCommentListLockPrefix  = "lock:comment:list"
	RedisCommentReplyLockPrefix = "lock:comment:reply"
	RedisCommentItemExpireSecs  = 24 * 60 * 60
	RedisCommentIndexExpireSecs = 30 * 60
	RedisCommentLockExpireSecs  = 5
)

func BuildLikeUserKey(userID string) string {
	return RedisLikeUserPrefix + ":" + userID
}

func BuildCommentItemKey(commentID string) string {
	return RedisCommentItemPrefix + ":" + commentID
}

func BuildCommentListKey(scene string, contentID string) string {
	return RedisCommentListPrefix + ":" + scene + ":" + contentID
}

func BuildCommentReplyKey(rootID string) string {
	return RedisCommentReplyPrefix + ":" + rootID
}

func BuildCommentListLockKey(scene string, contentID string) string {
	return RedisCommentListLockPrefix + ":" + scene + ":" + contentID
}

func BuildCommentReplyLockKey(rootID string) string {
	return RedisCommentReplyLockPrefix + ":" + rootID
}
