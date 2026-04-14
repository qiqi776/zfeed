package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/do"
)

const (
	LikeStatusLike   int32 = 10
	LikeStatusCancel int32 = 20
)

type LikeRepository interface {
	Upsert(likeDO *do.LikeDO) error
	CountByContentID(contentID int64) (int64, error)
	CountByContentIDs(contentIDs []int64) (map[int64]int64, error)
	IsLiked(userID int64, contentID int64) (bool, error)
	BatchIsLiked(userID int64, contentIDs []int64) (map[int64]bool, error)
}

type likeRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	logx.Logger
}

func NewLikeRepository(ctx context.Context, db *gorm.DB) LikeRepository {
	return &likeRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *likeRepositoryImpl) Upsert(likeDO *do.LikeDO) error {
	query := `
INSERT INTO zfeed_like (
  user_id,
  content_id,
  content_user_id,
  status,
  last_event_ts,
  is_deleted,
  created_by,
  updated_by
) VALUES (?, ?, ?, ?, ?, 0, ?, ?)
ON DUPLICATE KEY UPDATE
  status = IF(VALUES(last_event_ts) >= last_event_ts, VALUES(status), status),
  content_user_id = IF(VALUES(last_event_ts) >= last_event_ts AND VALUES(content_user_id) <> 0, VALUES(content_user_id), content_user_id),
  updated_by = IF(VALUES(last_event_ts) >= last_event_ts, VALUES(updated_by), updated_by),
  is_deleted = 0,
  last_event_ts = GREATEST(last_event_ts, VALUES(last_event_ts)),
  updated_at = IF(VALUES(last_event_ts) >= last_event_ts, CURRENT_TIMESTAMP, updated_at)
`

	return r.db.WithContext(r.ctx).Exec(
		query,
		likeDO.UserID,
		likeDO.ContentID,
		likeDO.ContentUserID,
		likeDO.Status,
		likeDO.LastEventTs,
		likeDO.CreatedBy,
		likeDO.UpdatedBy,
	).Error
}

func (r *likeRepositoryImpl) CountByContentID(contentID int64) (int64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	var count int64
	err := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Where("content_id = ? AND status = ? AND is_deleted = 0", contentID, LikeStatusLike).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *likeRepositoryImpl) CountByContentIDs(contentIDs []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(contentIDs))
	if len(contentIDs) == 0 {
		return result, nil
	}

	type row struct {
		ContentID int64 `gorm:"column:content_id"`
		Count     int64 `gorm:"column:count"`
	}

	rows := make([]row, 0)
	err := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Select("content_id, COUNT(*) AS count").
		Where("content_id IN ? AND status = ? AND is_deleted = 0", contentIDs, LikeStatusLike).
		Group("content_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, item := range rows {
		result[item.ContentID] = item.Count
	}
	return result, nil
}

func (r *likeRepositoryImpl) IsLiked(userID int64, contentID int64) (bool, error) {
	if userID <= 0 || contentID <= 0 {
		return false, nil
	}

	var count int64
	err := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Where("user_id = ? AND content_id = ? AND status = ? AND is_deleted = 0", userID, contentID, LikeStatusLike).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *likeRepositoryImpl) BatchIsLiked(userID int64, contentIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(contentIDs))
	if userID <= 0 || len(contentIDs) == 0 {
		return result, nil
	}

	type row struct {
		ContentID int64 `gorm:"column:content_id"`
	}

	rows := make([]row, 0)
	err := r.db.WithContext(r.ctx).
		Table("zfeed_like").
		Select("content_id").
		Where("user_id = ? AND content_id IN ? AND status = ? AND is_deleted = 0", userID, contentIDs, LikeStatusLike).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, item := range rows {
		result[item.ContentID] = true
	}
	return result, nil
}
