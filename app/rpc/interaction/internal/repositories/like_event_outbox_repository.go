package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	likeEventOutboxStatusPending = 10
	likeEventOutboxStatusSent    = 20
	maxLikeEventOutboxErrorLen   = 512
)

type LikeEventOutboxRecord struct {
	EventID    string
	EventType  string
	Payload    string
	RetryCount int
}

type LikeEventOutboxRepository interface {
	InsertPending(eventID, eventType, payload string, now time.Time) error
	ListDuePending(limit int, now time.Time) ([]LikeEventOutboxRecord, error)
	MarkSent(eventID string, sentAt time.Time) error
	MarkRetry(eventID string, nextRetryAt time.Time, lastError string) error
}

type likeEventOutboxRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	logx.Logger
}

type likeEventOutbox struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	EventID     string     `gorm:"column:event_id"`
	EventType   string     `gorm:"column:event_type"`
	Payload     string     `gorm:"column:payload"`
	Status      int32      `gorm:"column:status"`
	RetryCount  int32      `gorm:"column:retry_count"`
	NextRetryAt time.Time  `gorm:"column:next_retry_at"`
	LastError   string     `gorm:"column:last_error"`
	SentAt      *time.Time `gorm:"column:sent_at"`
}

func (likeEventOutbox) TableName() string {
	return "zfeed_like_event_outbox"
}

func NewLikeEventOutboxRepository(ctx context.Context, db *gorm.DB) LikeEventOutboxRepository {
	return &likeEventOutboxRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *likeEventOutboxRepositoryImpl) InsertPending(eventID, eventType, payload string, now time.Time) error {
	if eventID == "" || eventType == "" || payload == "" {
		return fmt.Errorf("invalid like event outbox record")
	}

	record := &likeEventOutbox{
		EventID:     eventID,
		EventType:   eventType,
		Payload:     payload,
		Status:      likeEventOutboxStatusPending,
		RetryCount:  0,
		NextRetryAt: now,
		LastError:   "",
		SentAt:      nil,
	}

	return r.db.WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(record).Error
}

func (r *likeEventOutboxRepositoryImpl) ListDuePending(limit int, now time.Time) ([]LikeEventOutboxRecord, error) {
	if limit <= 0 {
		return nil, nil
	}

	rows := make([]likeEventOutbox, 0, limit)
	err := r.db.WithContext(r.ctx).
		Table("zfeed_like_event_outbox").
		Select("event_id", "event_type", "payload", "retry_count").
		Where("status = ? AND next_retry_at <= ?", likeEventOutboxStatusPending, now).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]LikeEventOutboxRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, LikeEventOutboxRecord{
			EventID:    row.EventID,
			EventType:  row.EventType,
			Payload:    row.Payload,
			RetryCount: int(row.RetryCount),
		})
	}
	return result, nil
}

func (r *likeEventOutboxRepositoryImpl) MarkSent(eventID string, sentAt time.Time) error {
	if eventID == "" {
		return nil
	}

	return r.db.WithContext(r.ctx).
		Table("zfeed_like_event_outbox").
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":        likeEventOutboxStatusSent,
			"sent_at":       sentAt,
			"last_error":    "",
			"next_retry_at": sentAt,
		}).Error
}

func (r *likeEventOutboxRepositoryImpl) MarkRetry(eventID string, nextRetryAt time.Time, lastError string) error {
	if eventID == "" {
		return nil
	}

	return r.db.WithContext(r.ctx).
		Table("zfeed_like_event_outbox").
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":        likeEventOutboxStatusPending,
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": nextRetryAt,
			"last_error":    truncateLikeEventOutboxError(lastError),
			"sent_at":       nil,
		}).Error
}

func truncateLikeEventOutboxError(lastError string) string {
	if len(lastError) <= maxLikeEventOutboxErrorLen {
		return lastError
	}
	return lastError[:maxLikeEventOutboxErrorLen]
}
