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
	FollowStatusFollow   int32 = 10
	FollowStatusUnfollow int32 = 20
)

type FollowRepository interface {
	WithTx(tx *gorm.DB) FollowRepository
	Upsert(followDO *do.FollowDO) error
	GetByUserAndFollow(userID, followUserID int64) (*do.FollowDO, error)
	IsFollowing(userID, followUserID int64) (bool, error)
	CountFollowees(userID int64) (int64, error)
	CountFollowers(userID int64) (int64, error)
	ListFolloweesByCursor(userID int64, cursorFollowUserID int64, limit int) ([]int64, error)
}

type followRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewFollowRepository(ctx context.Context, db *gorm.DB) FollowRepository {
	return &followRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *followRepositoryImpl) WithTx(tx *gorm.DB) FollowRepository {
	return &followRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *followRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *followRepositoryImpl) Upsert(followDO *do.FollowDO) error {
	row := &model.ZfeedFollow{
		UserID:       followDO.UserID,
		FollowUserID: followDO.FollowUserID,
		Status:       followDO.Status,
		Version:      1,
		IsDeleted:    0,
		CreatedBy:    followDO.CreatedBy,
		UpdatedBy:    followDO.UpdatedBy,
	}

	return r.getDB().WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "follow_user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"status":     row.Status,
				"is_deleted": 0,
				"updated_by": row.UpdatedBy,
			}),
		}).
		Create(row).Error
}

func (r *followRepositoryImpl) GetByUserAndFollow(userID, followUserID int64) (*do.FollowDO, error) {
	var row model.ZfeedFollow
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFollow{}).
		Where("user_id = ? AND follow_user_id = ?", userID, followUserID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &do.FollowDO{
		UserID:       row.UserID,
		FollowUserID: row.FollowUserID,
		Status:       row.Status,
		CreatedBy:    row.CreatedBy,
		UpdatedBy:    row.UpdatedBy,
	}, nil
}

func (r *followRepositoryImpl) IsFollowing(userID, followUserID int64) (bool, error) {
	if userID <= 0 || followUserID <= 0 {
		return false, nil
	}

	row, err := r.GetByUserAndFollow(userID, followUserID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	return row.Status == FollowStatusFollow, nil
}

func (r *followRepositoryImpl) CountFollowees(userID int64) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}

	var count int64
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFollow{}).
		Where("user_id = ? AND status = ? AND is_deleted = 0", userID, FollowStatusFollow).
		Count(&count).Error
	return count, err
}

func (r *followRepositoryImpl) CountFollowers(userID int64) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}

	var count int64
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFollow{}).
		Where("follow_user_id = ? AND status = ? AND is_deleted = 0", userID, FollowStatusFollow).
		Count(&count).Error
	return count, err
}

func (r *followRepositoryImpl) ListFolloweesByCursor(userID int64, cursorFollowUserID int64, limit int) ([]int64, error) {
	if userID <= 0 || limit <= 0 {
		return []int64{}, nil
	}

	query := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedFollow{}).
		Select("follow_user_id").
		Where("user_id = ? AND status = ? AND is_deleted = 0", userID, FollowStatusFollow)

	if cursorFollowUserID > 0 {
		query = query.Where("follow_user_id < ?", cursorFollowUserID)
	}

	rows := make([]*model.ZfeedFollow, 0, limit)
	if err := query.Order("follow_user_id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.FollowUserID <= 0 {
			continue
		}
		ids = append(ids, row.FollowUserID)
	}
	return ids, nil
}
