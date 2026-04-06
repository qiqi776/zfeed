package model

import "time"

type ZfeedFavorite struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID        int64     `gorm:"column:user_id;uniqueIndex:uk_user_content"`
	Status        int32     `gorm:"column:status"`
	ContentID     int64     `gorm:"column:content_id;uniqueIndex:uk_user_content"`
	ContentUserID int64     `gorm:"column:content_user_id"`
	CreatedBy     int64     `gorm:"column:created_by"`
	UpdatedBy     int64     `gorm:"column:updated_by"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (ZfeedFavorite) TableName() string {
	return "zfeed_favorite"
}

type ZfeedFollow struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64     `gorm:"column:user_id;uniqueIndex:uk_user_follow_user"`
	FollowUserID int64     `gorm:"column:follow_user_id;uniqueIndex:uk_user_follow_user"`
	Status       int32     `gorm:"column:status"`
	Version      int32     `gorm:"column:version"`
	IsDeleted    int32     `gorm:"column:is_deleted"`
	CreatedBy    int64     `gorm:"column:created_by"`
	UpdatedBy    int64     `gorm:"column:updated_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (ZfeedFollow) TableName() string {
	return "zfeed_follow"
}
