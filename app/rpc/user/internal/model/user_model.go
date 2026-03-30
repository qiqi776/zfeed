package model

import "time"

type ZfeedUser struct {
    ID           int64      `gorm:"column:id;primaryKey;autoIncrement"`
    Username     string     `gorm:"column:username"`
    Nickname     string     `gorm:"column:nickname"`
    Avatar       string     `gorm:"column:avatar"`
    Bio          string     `gorm:"column:bio"`
    Mobile       string     `gorm:"column:mobile"`
    Email        string     `gorm:"column:email"`
    PasswordHash string     `gorm:"column:password_hash"`
    PasswordSalt string     `gorm:"column:password_salt"`
    Gender       int32      `gorm:"column:gender"`
    Birthday     *time.Time `gorm:"column:birthday"`
    Status       int32      `gorm:"column:status"`
    IsDeleted    int32      `gorm:"column:is_deleted"`
    CreatedBy    int64      `gorm:"column:created_by"`
    UpdatedBy    int64      `gorm:"column:updated_by"`
    CreatedAt    time.Time  `gorm:"column:created_at"`
    UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (ZfeedUser) TableName() string {
	return "zfeed_user"
}