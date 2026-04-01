package do

import "time"

type ContentDO struct {
	ID            int64
	UserID        int64
	ContentType   int32
	Status        int32
	Visibility    int32
	LikeCount     int64
	FavoriteCount int64
	CommentCount  int64
	PublishedAt   *time.Time
	IsDeleted     int32
	CreatedBy     int64
	UpdatedBy     int64
}

type ArticleDO struct {
	ID          int64
	ContentID   int64
	Title       string
	Description *string
	Cover       string
	Content     string
	IsDeleted   int32
}

type VideoDO struct {
	ID              int64
	ContentID       int64
	Title           string
	Description     *string
	OriginURL       string
	CoverURL        string
	Duration        int32
	TranscodeStatus int32
	IsDeleted       int32
}
