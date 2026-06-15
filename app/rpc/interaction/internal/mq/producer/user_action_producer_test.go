package producer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/mq/event"
)

type userActionOutboxRow struct {
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

func (userActionOutboxRow) TableName() string {
	return "zfeed_user_action_outbox"
}

func TestUserActionOutboxMetricLabelsExcludeHighCardinalityIDs(t *testing.T) {
	for _, label := range userActionOutboxMetricLabels {
		switch label {
		case "user_id", "content_id", "target_id", "event_id":
			t.Fatalf("user-action outbox metric must not include high-cardinality label %q", label)
		}
	}
}

func TestUserActionOutboxMetricNormalizersClampUnknownValues(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		in   string
		want string
	}{
		{
			name: "action allowed",
			fn:   normalizeUserActionOutboxActionLabel,
			in:   " Favorite ",
			want: event.UserActionFavorite,
		},
		{
			name: "action unknown",
			fn:   normalizeUserActionOutboxActionLabel,
			in:   "user_1001",
			want: userActionOutboxLabelUnknown,
		},
		{
			name: "result allowed",
			fn:   normalizeUserActionOutboxResultLabel,
			in:   " Mark-Failed ",
			want: userActionOutboxResultMarkFailed,
		},
		{
			name: "result unknown",
			fn:   normalizeUserActionOutboxResultLabel,
			in:   "content_2001",
			want: userActionOutboxLabelUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.in); got != tt.want {
				t.Fatalf("normalizer(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSendUserActionPersistsPendingOutboxOnPushFailure(t *testing.T) {
	t.Parallel()

	db := newUserActionProducerTestDB(t)
	pusher := &fakePusher{err: errors.New("kafka unavailable")}
	producer := &UserActionProducer{
		pusher:     pusher,
		db:         db,
		maxRetries: 1,
	}

	err := producer.SendUserAction(context.Background(), event.UserActionEvent{
		Action:        event.UserActionFavorite,
		UserID:        1001,
		TargetType:    event.UserActionTargetContent,
		TargetID:      2001,
		ContentUserID: 3001,
		Scene:         "ARTICLE",
	})
	if err != nil {
		t.Fatalf("SendUserAction returned error: %v", err)
	}

	var row userActionOutboxRow
	if err := db.Table("zfeed_user_action_outbox").First(&row).Error; err != nil {
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
	if !strings.Contains(row.Payload, `"action":"favorite"`) ||
		!strings.Contains(row.Payload, `"target_id":2001`) ||
		!strings.Contains(row.Payload, `"source":"interaction"`) {
		t.Fatalf("outbox payload = %s, want user action payload", row.Payload)
	}
	if len(pusher.payloads) != 1 {
		t.Fatalf("push attempts = %d, want 1", len(pusher.payloads))
	}
}

func TestSendUserActionRecordsOutboxMetrics(t *testing.T) {
	tests := []struct {
		name       string
		pushErr    error
		wantResult string
	}{
		{
			name:       "sent",
			wantResult: userActionOutboxResultSent,
		},
		{
			name:       "retry",
			pushErr:    errors.New("kafka unavailable"),
			wantResult: userActionOutboxResultRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRecord := recordUserActionOutboxMetric
			defer func() {
				recordUserActionOutboxMetric = oldRecord
			}()

			records := []struct {
				action string
				result string
			}{}
			recordUserActionOutboxMetric = func(action, result string) {
				records = append(records, struct {
					action string
					result string
				}{action: action, result: result})
			}

			producer := &UserActionProducer{
				pusher:     &fakePusher{err: tt.pushErr},
				db:         newUserActionProducerTestDB(t),
				maxRetries: 1,
			}

			err := producer.SendUserAction(context.Background(), event.UserActionEvent{
				Action:        event.UserActionFavorite,
				UserID:        1001,
				TargetType:    event.UserActionTargetContent,
				TargetID:      2001,
				ContentUserID: 3001,
				Scene:         "ARTICLE",
			})
			if err != nil {
				t.Fatalf("SendUserAction returned error: %v", err)
			}

			if len(records) != 1 {
				t.Fatalf("metric records = %+v, want one record", records)
			}
			if records[0].action != event.UserActionFavorite || records[0].result != tt.wantResult {
				t.Fatalf("metric record = %+v, want favorite/%s", records[0], tt.wantResult)
			}
		})
	}
}

func TestDispatchDueUserActionsMarksOutboxSentOnSuccess(t *testing.T) {
	t.Parallel()

	db := newUserActionProducerTestDB(t)
	if err := db.Create(&userActionOutboxRow{
		EventID:     "ua-evt-1",
		EventType:   "favorite",
		Payload:     `{"event_id":"ua-evt-1","action":"favorite","target_id":2001}`,
		Status:      10,
		RetryCount:  1,
		NextRetryAt: time.Now().Add(-time.Minute),
		LastError:   "previous failure",
	}).Error; err != nil {
		t.Fatalf("seed outbox row: %v", err)
	}

	pusher := &fakePusher{}
	producer := &UserActionProducer{
		pusher:     pusher,
		db:         db,
		maxRetries: 1,
	}

	producer.dispatchDueEvents(context.Background(), 10)

	var row userActionOutboxRow
	if err := db.Table("zfeed_user_action_outbox").Where("event_id = ?", "ua-evt-1").First(&row).Error; err != nil {
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

func TestDispatchDueUserActionsRecordsOutboxMetrics(t *testing.T) {
	tests := []struct {
		name       string
		pushErr    error
		wantResult string
	}{
		{
			name:       "replayed",
			wantResult: userActionOutboxResultReplayed,
		},
		{
			name:       "retry",
			pushErr:    errors.New("kafka unavailable"),
			wantResult: userActionOutboxResultRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRecord := recordUserActionOutboxMetric
			defer func() {
				recordUserActionOutboxMetric = oldRecord
			}()

			records := []struct {
				action string
				result string
			}{}
			recordUserActionOutboxMetric = func(action, result string) {
				records = append(records, struct {
					action string
					result string
				}{action: action, result: result})
			}

			db := newUserActionProducerTestDB(t)
			if err := db.Create(&userActionOutboxRow{
				EventID:     "ua-evt-metric",
				EventType:   event.UserActionComment,
				Payload:     `{"event_id":"ua-evt-metric","action":"comment","target_id":2001}`,
				Status:      10,
				RetryCount:  1,
				NextRetryAt: time.Now().Add(-time.Minute),
			}).Error; err != nil {
				t.Fatalf("seed outbox row: %v", err)
			}

			producer := &UserActionProducer{
				pusher:     &fakePusher{err: tt.pushErr},
				db:         db,
				maxRetries: 1,
			}
			producer.dispatchDueEvents(context.Background(), 10)

			if len(records) != 1 {
				t.Fatalf("metric records = %+v, want one record", records)
			}
			if records[0].action != event.UserActionComment || records[0].result != tt.wantResult {
				t.Fatalf("metric record = %+v, want comment/%s", records[0], tt.wantResult)
			}
		})
	}
}

func newUserActionProducerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
CREATE TABLE zfeed_user_action_outbox (
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
