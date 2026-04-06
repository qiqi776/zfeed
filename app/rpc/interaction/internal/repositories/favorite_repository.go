package repositories

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/model"
)

const (
	FavoriteStatusActive int32 = 10
)

type FavoriteRepository interface {
	WithTx(tx *gorm.DB) FavoriteRepository
	CountByContentID(contentID int64) (int64, error)
	IsFavorited(userID int64, contentID int64) (bool, error)
	Upsert(favoriteDO *do.FavoriteDO) error
	DeleteByUserAndContent(userID int64, contentID int64) (bool, error)
	ListByUserCursor(userID int64, cursor int64, limit int) ([]*model.ZfeedFavorite, error)
	GetByUserAndContent(userID int64, contentID int64) (*model.ZfeedFavorite, error)
}

type favoriteRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewFavoriteRepository(ctx context.Context, db *gorm.DB) FavoriteRepository {
	return &favoriteRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *favoriteRepositoryImpl) WithTx(tx *gorm.DB) FavoriteRepository {
	return &favoriteRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *favoriteRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *favoriteRepositoryImpl) CountByContentID(contentID int64) (int64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	var count int64
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("content_id = ? AND status = ?", contentID, FavoriteStatusActive).
		Count(&count).Error
	return count, err
}

func (r *favoriteRepositoryImpl) IsFavorited(userID int64, contentID int64) (bool, error) {
	if userID <= 0 || contentID <= 0 {
		return false, nil
	}

	var count int64
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("user_id = ? AND content_id = ? AND status = ?", userID, contentID, FavoriteStatusActive).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *favoriteRepositoryImpl) Upsert(favoriteDO *do.FavoriteDO) error {
	row := &model.ZfeedFavorite{
		UserID:        favoriteDO.UserID,
		Status:        favoriteDO.Status,
		ContentID:     favoriteDO.ContentID,
		ContentUserID: favoriteDO.ContentUserID,
		CreatedBy:     favoriteDO.CreatedBy,
		UpdatedBy:     favoriteDO.UpdatedBy,
	}

	return r.getDB().WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "content_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"status":          row.Status,
				"content_user_id": row.ContentUserID,
				"updated_by":      row.UpdatedBy,
			}),
		}).
		Create(row).Error
}

func (r *favoriteRepositoryImpl) DeleteByUserAndContent(userID int64, contentID int64) (bool, error) {
	tx := r.getDB().WithContext(r.ctx).
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Delete(&model.ZfeedFavorite{})
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *favoriteRepositoryImpl) ListByUserCursor(userID int64, cursor int64, limit int) ([]*model.ZfeedFavorite, error) {
	if userID <= 0 {
		return []*model.ZfeedFavorite{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	query := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("user_id = ? AND status = ?", userID, FavoriteStatusActive)

	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	rows := make([]*model.ZfeedFavorite, 0, limit)
	err := query.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *favoriteRepositoryImpl) GetByUserAndContent(userID int64, contentID int64) (*model.ZfeedFavorite, error) {
	var row model.ZfeedFavorite
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("user_id = ? AND content_id = ? AND status = ?", userID, contentID, FavoriteStatusActive).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}
