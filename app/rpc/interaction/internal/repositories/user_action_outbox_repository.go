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
	userActionOutboxStatusPending = 10
	userActionOutboxStatusSent    = 20
	maxUserActionOutboxErrorLen   = 512
)

type UserActionOutboxRecord struct {
	EventID    string
	EventType  string
	Payload    string
	RetryCount int
}

type UserActionOutboxRepository interface {
	InsertPending(eventID, eventType, payload string, now time.Time) error
	ListDuePending(limit int, now time.Time) ([]UserActionOutboxRecord, error)
	MarkSent(eventID string, sentAt time.Time) error
	MarkRetry(eventID string, nextRetryAt time.Time, lastError string) error
}

type userActionOutboxRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	logx.Logger
}

type userActionOutbox struct {
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

func (userActionOutbox) TableName() string {
	return "zfeed_user_action_outbox"
}

func NewUserActionOutboxRepository(ctx context.Context, db *gorm.DB) UserActionOutboxRepository {
	return &userActionOutboxRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *userActionOutboxRepositoryImpl) InsertPending(eventID, eventType, payload string, now time.Time) error {
	if eventID == "" || eventType == "" || payload == "" {
		return fmt.Errorf("invalid user action outbox record")
	}

	record := &userActionOutbox{
		EventID:     eventID,
		EventType:   eventType,
		Payload:     payload,
		Status:      userActionOutboxStatusPending,
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

func (r *userActionOutboxRepositoryImpl) ListDuePending(limit int, now time.Time) ([]UserActionOutboxRecord, error) {
	if limit <= 0 {
		return []UserActionOutboxRecord{}, nil
	}

	rows := make([]userActionOutbox, 0, limit)
	err := r.db.WithContext(r.ctx).
		Table("zfeed_user_action_outbox").
		Select("event_id", "event_type", "payload", "retry_count").
		Where("status = ? AND next_retry_at <= ?", userActionOutboxStatusPending, now).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]UserActionOutboxRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, UserActionOutboxRecord{
			EventID:    row.EventID,
			EventType:  row.EventType,
			Payload:    row.Payload,
			RetryCount: int(row.RetryCount),
		})
	}
	return result, nil
}

func (r *userActionOutboxRepositoryImpl) MarkSent(eventID string, sentAt time.Time) error {
	if eventID == "" {
		return nil
	}

	return r.db.WithContext(r.ctx).
		Table("zfeed_user_action_outbox").
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":        userActionOutboxStatusSent,
			"sent_at":       sentAt,
			"last_error":    "",
			"next_retry_at": sentAt,
		}).Error
}

func (r *userActionOutboxRepositoryImpl) MarkRetry(eventID string, nextRetryAt time.Time, lastError string) error {
	if eventID == "" {
		return nil
	}

	return r.db.WithContext(r.ctx).
		Table("zfeed_user_action_outbox").
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":        userActionOutboxStatusPending,
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": nextRetryAt,
			"last_error":    truncateUserActionOutboxError(lastError),
			"sent_at":       nil,
		}).Error
}

func truncateUserActionOutboxError(lastError string) string {
	if len(lastError) <= maxUserActionOutboxErrorLen {
		return lastError
	}
	return lastError[:maxUserActionOutboxErrorLen]
}
