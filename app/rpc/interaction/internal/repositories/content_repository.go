package repositories

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ContentRepository interface {
	GetAuthorID(contentID int64) (int64, error)
}

type contentRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	logx.Logger
}

func NewContentRepository(ctx context.Context, db *gorm.DB) ContentRepository {
	return &contentRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *contentRepositoryImpl) GetAuthorID(contentID int64) (int64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	var row struct {
		UserID int64 `gorm:"column:user_id"`
	}

	err := r.db.WithContext(r.ctx).
		Table("zfeed_content").
		Select("user_id").
		Where("id = ? AND is_deleted = 0", contentID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return row.UserID, nil
}
