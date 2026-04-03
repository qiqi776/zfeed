package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MqConsumeDedupRepository interface {
	InsertIfAbsent(consumer, eventID string) (bool, error)
}

type mqConsumeDedupRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	logx.Logger
}

type mqConsumeDedup struct {
	ID       uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	Consumer string `gorm:"column:consumer"`
	EventID  string `gorm:"column:event_id"`
}

func (mqConsumeDedup) TableName() string {
	return "zfeed_mq_consume_dedup"
}

func NewMqConsumeDedupRepository(ctx context.Context, db *gorm.DB) MqConsumeDedupRepository {
	return &mqConsumeDedupRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *mqConsumeDedupRepositoryImpl) InsertIfAbsent(consumer, eventID string) (bool, error) {
	if consumer == "" || eventID == "" {
		return false, nil
	}

	record := &mqConsumeDedup{
		Consumer: consumer,
		EventID:  eventID,
	}

	tx := r.db.WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "consumer"}, {Name: "event_id"}},
			DoNothing: true,
		}).
		Create(record)
	if tx.Error != nil {
		return false, tx.Error
	}

	return tx.RowsAffected == 1, nil
}
