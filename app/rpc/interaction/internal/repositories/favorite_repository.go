package repositories

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/model"
)

const (
	FavoriteStatusActive int32 = 10
)

type FavoriteRepository interface {
	WithTx(tx *gorm.DB) FavoriteRepository
	CountByTarget(scene int32, contentID int64) (int64, error)
	IsFavorited(userID int64, scene int32, contentID int64) (bool, error)
	Upsert(favoriteDO *do.FavoriteDO) error
	DeleteByUserAndTarget(userID int64, scene int32, contentID int64) (bool, error)
	ListContentByUserCursor(userID int64, cursor int64, limit int) ([]*model.ZfeedFavorite, error)
	GetByUserAndTarget(userID int64, scene int32, contentID int64) (*model.ZfeedFavorite, error)
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

func (r *favoriteRepositoryImpl) CountByTarget(scene int32, contentID int64) (int64, error) {
	if contentID <= 0 || scene <= 0 {
		return 0, nil
	}

	var count int64
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("scene = ? AND content_id = ? AND status = ?", scene, contentID, FavoriteStatusActive).
		Count(&count).Error
	return count, err
}

func (r *favoriteRepositoryImpl) IsFavorited(userID int64, scene int32, contentID int64) (bool, error) {
	if userID <= 0 || contentID <= 0 || scene <= 0 {
		return false, nil
	}

	var count int64
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("user_id = ? AND scene = ? AND content_id = ? AND status = ?", userID, scene, contentID, FavoriteStatusActive).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *favoriteRepositoryImpl) Upsert(favoriteDO *do.FavoriteDO) error {
	row := &model.ZfeedFavorite{
		UserID:        favoriteDO.UserID,
		Scene:         favoriteDO.Scene,
		Status:        favoriteDO.Status,
		ContentID:     favoriteDO.ContentID,
		ContentUserID: favoriteDO.ContentUserID,
		CreatedBy:     favoriteDO.CreatedBy,
		UpdatedBy:     favoriteDO.UpdatedBy,
	}

	return r.getDB().WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "scene"}, {Name: "content_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"status":          row.Status,
				"content_user_id": row.ContentUserID,
				"updated_by":      row.UpdatedBy,
			}),
		}).
		Create(row).Error
}

func (r *favoriteRepositoryImpl) DeleteByUserAndTarget(userID int64, scene int32, contentID int64) (bool, error) {
	tx := r.getDB().WithContext(r.ctx).
		Where("user_id = ? AND scene = ? AND content_id = ?", userID, scene, contentID).
		Delete(&model.ZfeedFavorite{})
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *favoriteRepositoryImpl) ListContentByUserCursor(userID int64, cursor int64, limit int) ([]*model.ZfeedFavorite, error) {
	if userID <= 0 {
		return []*model.ZfeedFavorite{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	query := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("user_id = ? AND status = ? AND scene IN ?", userID, FavoriteStatusActive, []int32{
			int32(interaction.Scene_ARTICLE),
			int32(interaction.Scene_VIDEO),
		})

	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	rows := make([]*model.ZfeedFavorite, 0, limit)
	err := query.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *favoriteRepositoryImpl) GetByUserAndTarget(userID int64, scene int32, contentID int64) (*model.ZfeedFavorite, error) {
	var row model.ZfeedFavorite
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFavorite{}).
		Where("user_id = ? AND scene = ? AND content_id = ? AND status = ?", userID, scene, contentID, FavoriteStatusActive).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}
