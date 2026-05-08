package producer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakePusher struct {
	err      error
	payloads []string
}

func (p *fakePusher) Push(_ context.Context, value string) error {
	p.payloads = append(p.payloads, value)
	return p.err
}

type likeOutboxRow struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement"`
	EventID     string     `gorm:"column:event_id"`
	EventType   string     `gorm:"column:event_type"`
	Payload     string     `gorm:"column:payload"`
	Status      int32      `gorm:"column:status"`
	RetryCount  int32      `gorm:"column:retry_count"`
	NextRetryAt time.Time  `gorm:"column:next_retry_at"`
	LastError   string     `gorm:"column:last_error"`
	SentAt      *time.Time `gorm:"column:sent_at"`
}

func (likeOutboxRow) TableName() string {
	return "zfeed_like_event_outbox"
}

func TestSendLikeEventPersistsPendingOutboxOnPushFailure(t *testing.T) {
	t.Parallel()

	db := newLikeProducerTestDB(t)
	pusher := &fakePusher{err: errors.New("kafka unavailable")}
	producer := &LikeProducer{
		pusher:     pusher,
		db:         db,
		maxRetries: 1,
	}

	if err := producer.SendLikeEvent(context.Background(), 1001, 9001, 2001, "ARTICLE"); err != nil {
		t.Fatalf("SendLikeEvent returned error: %v", err)
	}

	var row likeOutboxRow
	if err := db.Table("zfeed_like_event_outbox").First(&row).Error; err != nil {
		t.Fatalf("load outbox row: %v", err)
	}
	if row.Status != 10 {
		t.Fatalf("outbox status = %d, want 10", row.Status)
	}
	if row.RetryCount != 1 {
		t.Fatalf("outbox retry_count = %d, want 1", row.RetryCount)
	}
	if row.SentAt != nil {
		t.Fatalf("outbox sent_at = %v, want nil", row.SentAt)
	}
	if !strings.Contains(row.Payload, "\"event_type\":\"like\"") {
		t.Fatalf("outbox payload = %s, want like event payload", row.Payload)
	}
	if len(pusher.payloads) != 1 {
		t.Fatalf("push attempts = %d, want 1", len(pusher.payloads))
	}
}

func TestDispatchDueEventsMarksOutboxSentOnSuccess(t *testing.T) {
	t.Parallel()

	db := newLikeProducerTestDB(t)
	if err := db.Create(&likeOutboxRow{
		EventID:     "evt-1",
		EventType:   "like",
		Payload:     `{"event_id":"evt-1","event_type":"like"}`,
		Status:      10,
		RetryCount:  1,
		NextRetryAt: time.Now().Add(-time.Minute),
		LastError:   "previous failure",
	}).Error; err != nil {
		t.Fatalf("seed outbox row: %v", err)
	}

	pusher := &fakePusher{}
	producer := &LikeProducer{
		pusher:     pusher,
		db:         db,
		maxRetries: 1,
	}

	producer.dispatchDueEvents(context.Background(), 10)

	var row likeOutboxRow
	if err := db.Table("zfeed_like_event_outbox").Where("event_id = ?", "evt-1").First(&row).Error; err != nil {
		t.Fatalf("load outbox row: %v", err)
	}
	if row.Status != 20 {
		t.Fatalf("outbox status = %d, want 20", row.Status)
	}
	if row.SentAt == nil {
		t.Fatal("outbox sent_at = nil, want non-nil")
	}
	if row.RetryCount != 1 {
		t.Fatalf("outbox retry_count = %d, want keep 1", row.RetryCount)
	}
	if len(pusher.payloads) != 1 {
		t.Fatalf("push attempts = %d, want 1", len(pusher.payloads))
	}
}

func newLikeProducerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
CREATE TABLE zfeed_like_event_outbox (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  status INTEGER NOT NULL DEFAULT 10,
  retry_count INTEGER NOT NULL DEFAULT 0,
  next_retry_at DATETIME NOT NULL,
  last_error TEXT NOT NULL DEFAULT '',
  sent_at DATETIME NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(event_id)
)`).Error; err != nil {
		t.Fatalf("create outbox table: %v", err)
	}
	return db
}
