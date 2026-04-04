package consumer

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/mq/event"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/app/rpc/interaction/internal/testutil/mysqltest"
)

const (
	likeConsumerTestMinID = int64(901000)
	likeConsumerTestMaxID = int64(901999)
	likeConsumerName      = "interaction.like_consumer"
)

type consumedLikeRow struct {
	UserID        int64 `gorm:"column:user_id"`
	ContentID     int64 `gorm:"column:content_id"`
	ContentUserID int64 `gorm:"column:content_user_id"`
	Status        int32 `gorm:"column:status"`
	LastEventTs   int64 `gorm:"column:last_event_ts"`
}

func TestLikeConsumerDedupsSameEvent(t *testing.T) {
	eventPrefix := "like-consumer-dup-like-"
	db, cleanup := openLikeConsumerTestDB(t, eventPrefix)
	defer cleanup()

	consumer := NewLikeConsumer(context.Background(), &svc.ServiceContext{MysqlDb: db})
	raw := mustMarshalLikeEvent(t, &event.LikeEvent{
		EventID:       eventPrefix + "evt-1",
		EventType:     event.EventTypeLike,
		UserID:        likeConsumerTestMinID + 1,
		ContentID:     likeConsumerTestMinID + 101,
		ContentUserID: likeConsumerTestMinID + 201,
		Scene:         "ARTICLE",
		Timestamp:     100,
	})

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("first consume returned error: %v", err)
	}
	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("second consume returned error: %v", err)
	}

	assertDedupCount(t, db, eventPrefix, 1)
	assertLikeRowCount(t, db, likeConsumerTestMinID+1, likeConsumerTestMinID+101, 1)

	row := mustGetConsumedLikeRow(t, db, likeConsumerTestMinID+1, likeConsumerTestMinID+101)
	if row.Status != 10 {
		t.Fatalf("status = %d, want 10", row.Status)
	}
	if row.LastEventTs != 100 {
		t.Fatalf("last_event_ts = %d, want 100", row.LastEventTs)
	}
}

func TestLikeConsumerCoalescesDifferentEvents(t *testing.T) {
	eventPrefix := "like-consumer-distinct-events-"
	db, cleanup := openLikeConsumerTestDB(t, eventPrefix)
	defer cleanup()

	consumer := NewLikeConsumer(context.Background(), &svc.ServiceContext{MysqlDb: db})
	userID := likeConsumerTestMinID + 2
	contentID := likeConsumerTestMinID + 102
	contentUserID := likeConsumerTestMinID + 202

	first := mustMarshalLikeEvent(t, &event.LikeEvent{
		EventID:       eventPrefix + "evt-1",
		EventType:     event.EventTypeLike,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         "ARTICLE",
		Timestamp:     100,
	})
	second := mustMarshalLikeEvent(t, &event.LikeEvent{
		EventID:       eventPrefix + "evt-2",
		EventType:     event.EventTypeLike,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         "ARTICLE",
		Timestamp:     200,
	})

	if err := consumer.Consume(context.Background(), "", first); err != nil {
		t.Fatalf("first consume returned error: %v", err)
	}
	if err := consumer.Consume(context.Background(), "", second); err != nil {
		t.Fatalf("second consume returned error: %v", err)
	}

	assertDedupCount(t, db, eventPrefix, 2)
	assertLikeRowCount(t, db, userID, contentID, 1)

	row := mustGetConsumedLikeRow(t, db, userID, contentID)
	if row.Status != 10 {
		t.Fatalf("status = %d, want 10", row.Status)
	}
	if row.LastEventTs != 200 {
		t.Fatalf("last_event_ts = %d, want 200", row.LastEventTs)
	}
}

func TestLikeConsumerIgnoresStaleEvent(t *testing.T) {
	eventPrefix := "like-consumer-stale-"
	db, cleanup := openLikeConsumerTestDB(t, eventPrefix)
	defer cleanup()

	consumer := NewLikeConsumer(context.Background(), &svc.ServiceContext{MysqlDb: db})
	userID := likeConsumerTestMinID + 3
	contentID := likeConsumerTestMinID + 103
	contentUserID := likeConsumerTestMinID + 203

	newer := mustMarshalLikeEvent(t, &event.LikeEvent{
		EventID:       eventPrefix + "cancel-200",
		EventType:     event.EventTypeCancel,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         "ARTICLE",
		Timestamp:     200,
	})
	older := mustMarshalLikeEvent(t, &event.LikeEvent{
		EventID:       eventPrefix + "like-100",
		EventType:     event.EventTypeLike,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID + 1,
		Scene:         "ARTICLE",
		Timestamp:     100,
	})

	if err := consumer.Consume(context.Background(), "", newer); err != nil {
		t.Fatalf("consume newer event returned error: %v", err)
	}
	if err := consumer.Consume(context.Background(), "", older); err != nil {
		t.Fatalf("consume older event returned error: %v", err)
	}

	assertDedupCount(t, db, eventPrefix, 2)
	assertLikeRowCount(t, db, userID, contentID, 1)

	row := mustGetConsumedLikeRow(t, db, userID, contentID)
	if row.Status != 20 {
		t.Fatalf("status = %d, want 20", row.Status)
	}
	if row.LastEventTs != 200 {
		t.Fatalf("last_event_ts = %d, want 200", row.LastEventTs)
	}
	if row.ContentUserID != contentUserID {
		t.Fatalf("content_user_id = %d, want %d", row.ContentUserID, contentUserID)
	}
}

func TestLikeConsumerDedupsCancel(t *testing.T) {
	eventPrefix := "like-consumer-dup-cancel-"
	db, cleanup := openLikeConsumerTestDB(t, eventPrefix)
	defer cleanup()

	consumer := NewLikeConsumer(context.Background(), &svc.ServiceContext{MysqlDb: db})
	raw := mustMarshalLikeEvent(t, &event.LikeEvent{
		EventID:       eventPrefix + "evt-1",
		EventType:     event.EventTypeCancel,
		UserID:        likeConsumerTestMinID + 4,
		ContentID:     likeConsumerTestMinID + 104,
		ContentUserID: likeConsumerTestMinID + 204,
		Scene:         "ARTICLE",
		Timestamp:     300,
	})

	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("first consume returned error: %v", err)
	}
	if err := consumer.Consume(context.Background(), "", raw); err != nil {
		t.Fatalf("second consume returned error: %v", err)
	}

	assertDedupCount(t, db, eventPrefix, 1)
	assertLikeRowCount(t, db, likeConsumerTestMinID+4, likeConsumerTestMinID+104, 1)

	row := mustGetConsumedLikeRow(t, db, likeConsumerTestMinID+4, likeConsumerTestMinID+104)
	if row.Status != 20 {
		t.Fatalf("status = %d, want 20", row.Status)
	}
	if row.LastEventTs != 300 {
		t.Fatalf("last_event_ts = %d, want 300", row.LastEventTs)
	}
}

func openLikeConsumerTestDB(t *testing.T, eventPrefix string) (*gorm.DB, func()) {
	t.Helper()

	db, err := mysqltest.Open()
	if err != nil {
		t.Skipf("skip MySQL-backed like consumer test: %v", err)
	}

	if err := mysqltest.EnsureLikeTables(db); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("ensure test tables: %v", err)
	}

	if err := mysqltest.CleanupLikeRowsByRange(db, likeConsumerTestMinID, likeConsumerTestMaxID); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("cleanup like rows before run: %v", err)
	}
	if err := mysqltest.CleanupDedupRows(db, likeConsumerName, eventPrefix); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("cleanup dedup rows before run: %v", err)
	}

	return db, func() {
		if err := mysqltest.CleanupLikeRowsByRange(db, likeConsumerTestMinID, likeConsumerTestMaxID); err != nil {
			t.Fatalf("cleanup like rows after run: %v", err)
		}
		if err := mysqltest.CleanupDedupRows(db, likeConsumerName, eventPrefix); err != nil {
			t.Fatalf("cleanup dedup rows after run: %v", err)
		}
		if err := mysqltest.Close(db); err != nil {
			t.Fatalf("close db: %v", err)
		}
	}
}

func mustMarshalLikeEvent(t *testing.T, evt *event.LikeEvent) string {
	t.Helper()

	body, err := evt.Marshal()
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	return string(body)
}

func assertDedupCount(t *testing.T, db *gorm.DB, eventPrefix string, want int64) {
	t.Helper()

	var count int64
	if err := db.Table("zfeed_mq_consume_dedup").
		Where("consumer = ? AND event_id LIKE ?", likeConsumerName, eventPrefix+"%").
		Count(&count).Error; err != nil {
		t.Fatalf("count dedup rows: %v", err)
	}
	if count != want {
		t.Fatalf("dedup row count = %d, want %d", count, want)
	}
}

func assertLikeRowCount(t *testing.T, db *gorm.DB, userID, contentID int64, want int64) {
	t.Helper()

	var count int64
	if err := db.Table("zfeed_like").
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Count(&count).Error; err != nil {
		t.Fatalf("count like rows: %v", err)
	}
	if count != want {
		t.Fatalf("like row count = %d, want %d", count, want)
	}
}

func mustGetConsumedLikeRow(t *testing.T, db *gorm.DB, userID, contentID int64) consumedLikeRow {
	t.Helper()

	var row consumedLikeRow
	if err := db.Table("zfeed_like").
		Select("user_id", "content_id", "content_user_id", "status", "last_event_ts").
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Take(&row).Error; err != nil {
		t.Fatalf("query like row: %v", err)
	}

	return row
}
