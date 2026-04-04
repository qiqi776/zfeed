package repositories

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMqConsumeDedupRepositoryInsertIfAbsent(t *testing.T) {
	db, err := newDedupTestDB(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec(`
CREATE TABLE zfeed_mq_consume_dedup (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  consumer TEXT NOT NULL,
  event_id TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(consumer, event_id)
)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	repo := NewMqConsumeDedupRepository(context.Background(), db)

	inserted, err := repo.InsertIfAbsent("interaction.like_consumer", "evt-1")
	if err != nil {
		t.Fatalf("first insert returned error: %v", err)
	}
	if !inserted {
		t.Fatal("first insert inserted=false, want true")
	}

	inserted, err = repo.InsertIfAbsent("interaction.like_consumer", "evt-1")
	if err != nil {
		t.Fatalf("duplicate insert returned error: %v", err)
	}
	if inserted {
		t.Fatal("duplicate insert inserted=true, want false")
	}
}

func TestMqConsumeDedupRepositoryInsertIfAbsent_AllowsSameEventForDifferentConsumers(t *testing.T) {
	db, err := newDedupTestDB(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := createDedupTable(db); err != nil {
		t.Fatalf("create table: %v", err)
	}

	repo := NewMqConsumeDedupRepository(context.Background(), db)

	inserted, err := repo.InsertIfAbsent("interaction.like_consumer", "evt-same")
	if err != nil {
		t.Fatalf("first insert returned error: %v", err)
	}
	if !inserted {
		t.Fatal("first insert inserted=false, want true")
	}

	inserted, err = repo.InsertIfAbsent("interaction.favorite_consumer", "evt-same")
	if err != nil {
		t.Fatalf("second insert returned error: %v", err)
	}
	if !inserted {
		t.Fatal("second insert inserted=false, want true")
	}
}

func TestMqConsumeDedupRepositoryInsertIfAbsent_EmptyInputIsNoop(t *testing.T) {
	db, err := newDedupTestDB(t)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := createDedupTable(db); err != nil {
		t.Fatalf("create table: %v", err)
	}

	repo := NewMqConsumeDedupRepository(context.Background(), db)

	testCases := []struct {
		name     string
		consumer string
		eventID  string
	}{
		{
			name:     "empty consumer",
			consumer: "",
			eventID:  "evt-1",
		},
		{
			name:     "empty event id",
			consumer: "interaction.like_consumer",
			eventID:  "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			inserted, err := repo.InsertIfAbsent(tc.consumer, tc.eventID)
			if err != nil {
				t.Fatalf("InsertIfAbsent returned error: %v", err)
			}
			if inserted {
				t.Fatal("InsertIfAbsent inserted=true, want false")
			}
		})
	}
}

func newDedupTestDB(t *testing.T) (*gorm.DB, error) {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
}

func createDedupTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE zfeed_mq_consume_dedup (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  consumer TEXT NOT NULL,
  event_id TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(consumer, event_id)
)`).Error
}
