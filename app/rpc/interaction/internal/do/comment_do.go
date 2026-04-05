package do

import "time"

type CommentDO struct {
	ID            int64
	ContentID     int64
	ContentUserID int64
	UserID        int64
	ReplyToUserID int64
	ParentID      int64
	RootID        int64
	Comment       string
	Status        int32
	Version       int32
	ReplyCount    int64
	IsDeleted     int32
	CreatedBy     int64
	UpdatedBy     int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
