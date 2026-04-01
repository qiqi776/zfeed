package model

import "time"

type ZfeedContent struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID        int64      `gorm:"column:user_id"`
	ContentType   int32      `gorm:"column:content_type"`
	Status        int32      `gorm:"column:status"`
	Visibility    int32      `gorm:"column:visibility"`
	LikeCount     int64      `gorm:"column:like_count"`
	FavoriteCount int64      `gorm:"column:favorite_count"`
	CommentCount  int64      `gorm:"column:comment_count"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	IsDeleted     int32      `gorm:"column:is_deleted"`
	CreatedBy     int64      `gorm:"column:created_by"`
	UpdatedBy     int64      `gorm:"column:updated_by"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (ZfeedContent) TableName() string {
	return "zfeed_content"
}

type ZfeedArticle struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement"`
	ContentID   int64      `gorm:"column:content_id"`
	Title       string     `gorm:"column:title"`
	Description *string    `gorm:"column:description"`
	Cover       string     `gorm:"column:cover"`
	Content     string     `gorm:"column:content"`
	IsDeleted   int32      `gorm:"column:is_deleted"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (ZfeedArticle) TableName() string {
	return "zfeed_article"
}

type ZfeedVideo struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	ContentID       int64      `gorm:"column:content_id"`
	Title           string     `gorm:"column:title"`
	Description     *string    `gorm:"column:description"`
	OriginURL       string     `gorm:"column:origin_url"`
	CoverURL        string     `gorm:"column:cover_url"`
	Duration        int32      `gorm:"column:duration"`
	TranscodeStatus int32      `gorm:"column:transcode_status"`
	IsDeleted       int32      `gorm:"column:is_deleted"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (ZfeedVideo) TableName() string {
	return "zfeed_video"
}
