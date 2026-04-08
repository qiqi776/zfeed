package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zfeed/app/rpc/count/internal/model"
)

type CountValueRepository interface {
	WithTx(tx *gorm.DB) CountValueRepository
	Get(bizType int32, targetType int32, targetID int64) (*model.ZfeedCountValue, error)
	BatchGet(bizType int32, targetType int32, targetIDs []int64) (map[int64]*model.ZfeedCountValue, error)
	SumByOwner(bizType int32, targetType int32, ownerID int64) (int64, error)
	ApplyDelta(bizType int32, targetType int32, targetID int64, ownerID int64, delta int64, updatedAt time.Time) (int64, error)
}

type countValueRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewCountValueRepository(ctx context.Context, db *gorm.DB) CountValueRepository {
	return &countValueRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *countValueRepositoryImpl) WithTx(tx *gorm.DB) CountValueRepository {
	return &countValueRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *countValueRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *countValueRepositoryImpl) Get(bizType int32, targetType int32, targetID int64) (*model.ZfeedCountValue, error) {
	if bizType <= 0 || targetType <= 0 || targetID <= 0 {
		return nil, nil
	}

	var row model.ZfeedCountValue
	err := r.getDB().WithContext(r.ctx).
		Where("biz_type = ? AND target_type = ? AND target_id = ?", bizType, targetType, targetID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *countValueRepositoryImpl) BatchGet(bizType int32, targetType int32, targetIDs []int64) (map[int64]*model.ZfeedCountValue, error) {
	result := make(map[int64]*model.ZfeedCountValue, len(targetIDs))
	if bizType <= 0 || targetType <= 0 || len(targetIDs) == 0 {
		return result, nil
	}

	rows := make([]*model.ZfeedCountValue, 0, len(targetIDs))
	err := r.getDB().WithContext(r.ctx).
		Where("biz_type = ? AND target_type = ? AND target_id IN ?", bizType, targetType, targetIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row == nil {
			continue
		}
		result[row.TargetID] = row
	}
	return result, nil
}

func (r *countValueRepositoryImpl) SumByOwner(bizType int32, targetType int32, ownerID int64) (int64, error) {
	if bizType <= 0 || targetType <= 0 || ownerID <= 0 {
		return 0, nil
	}

	var sum int64
	err := r.getDB().WithContext(r.ctx).
		Model(&model.ZfeedCountValue{}).
		Select("COALESCE(SUM(value), 0)").
		Where("biz_type = ? AND target_type = ? AND owner_id = ?", bizType, targetType, ownerID).
		Scan(&sum).Error
	if err != nil {
		return 0, err
	}
	return sum, nil
}

func (r *countValueRepositoryImpl) ApplyDelta(
	bizType int32,
	targetType int32,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) (int64, error) {
	if bizType <= 0 || targetType <= 0 || targetID <= 0 || delta == 0 {
		return 0, nil
	}

	if r.tx != nil {
		return r.applyDeltaOnce(r.tx.WithContext(r.ctx), bizType, targetType, targetID, ownerID, delta, updatedAt)
	}

	var finalValue int64
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = r.db.WithContext(r.ctx).Transaction(func(tx *gorm.DB) error {
			var innerErr error
			finalValue, innerErr = r.applyDeltaOnce(tx, bizType, targetType, targetID, ownerID, delta, updatedAt)
			return innerErr
		})
		if err == nil {
			return finalValue, nil
		}
		if !isRetryableTxErr(err) || attempt == 2 {
			return 0, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}

	return 0, err
}

func (r *countValueRepositoryImpl) applyDeltaOnce(
	tx *gorm.DB,
	bizType int32,
	targetType int32,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) (int64, error) {
	var row model.ZfeedCountValue
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("biz_type = ? AND target_type = ? AND target_id = ?", bizType, targetType, targetID).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if delta < 0 {
			return 0, nil
		}
		row = model.ZfeedCountValue{
			BizType:    bizType,
			TargetType: targetType,
			TargetID:   targetID,
			Value:      delta,
			Version:    1,
			OwnerID:    ownerID,
			CreatedAt:  updatedAt,
			UpdatedAt:  updatedAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return 0, err
		}
		return row.Value, nil
	}
	if err != nil {
		return 0, err
	}

	nextValue := row.Value + delta
	if nextValue < 0 {
		nextValue = 0
	}
	updates := map[string]any{
		"value":      nextValue,
		"version":    row.Version + 1,
		"updated_at": updatedAt,
	}
	if row.OwnerID == 0 && ownerID > 0 {
		updates["owner_id"] = ownerID
	}

	if err := tx.Model(&model.ZfeedCountValue{}).
		Where("id = ?", row.ID).
		Updates(updates).Error; err != nil {
		return 0, err
	}

	return nextValue, nil
}

func isRetryableTxErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadlock found when trying to get lock") ||
		strings.Contains(msg, "lock wait timeout exceeded") ||
		strings.Contains(msg, "database is locked")
}
