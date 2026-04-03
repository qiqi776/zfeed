package repositories

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMqConsumeDedupRepositoryInsertIfAbsent(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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
