package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/count/internal/model"
)

type MqConsumeDedupRepository interface {
	WithTx(tx *gorm.DB) MqConsumeDedupRepository
	InsertIfAbsent(consumer, eventID string) (bool, error)
}

type mqConsumeDedupRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	tx  *gorm.DB
	logx.Logger
}

func NewMqConsumeDedupRepository(ctx context.Context, db *gorm.DB) MqConsumeDedupRepository {
	return &mqConsumeDedupRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *mqConsumeDedupRepositoryImpl) WithTx(tx *gorm.DB) MqConsumeDedupRepository {
	return &mqConsumeDedupRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *mqConsumeDedupRepositoryImpl) getDB() *gorm.DB {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *mqConsumeDedupRepositoryImpl) InsertIfAbsent(consumer, eventID string) (bool, error) {
	if consumer == "" || eventID == "" {
		return false, nil
	}

	record := &model.ZfeedMqConsumeDedup{
		Consumer: consumer,
		EventID:  eventID,
	}
	err := r.getDB().WithContext(r.ctx).Create(record).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return false, nil
	}
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "duplicate") || strings.Contains(errMsg, "unique constraint") {
		return false, nil
	}
	return false, err
}
