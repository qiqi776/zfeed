package track

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDailyAggregatorAggregatesEventsByDayVariantAndSource(t *testing.T) {
	db := newDailyMetricTestDB(t)
	aggregator := NewDailyAggregator(db)

	occurredAt := time.Date(2026, 6, 15, 10, 30, 0, 0, time.UTC).Unix()
	events := []Event{
		{EventType: EventTypeExposure, VariantID: "b", Source: "recommend", OccurredAt: occurredAt},
		{EventType: EventTypeClick, VariantID: "b", Source: "recommend", OccurredAt: occurredAt},
		{EventType: EventTypeDwell, VariantID: "b", Source: "recommend", DwellMs: 1500, OccurredAt: occurredAt},
		{EventType: EventTypeLike, VariantID: "b", Source: "recommend", OccurredAt: occurredAt},
		{EventType: EventTypeFavorite, VariantID: "b", Source: "recommend", OccurredAt: occurredAt},
		{EventType: EventTypeComment, VariantID: "b", Source: "recommend", OccurredAt: occurredAt},
	}

	for _, event := range events {
		if err := aggregator.Aggregate(context.Background(), event); err != nil {
			t.Fatalf("Aggregate(%s) returned error: %v", event.EventType, err)
		}
	}

	var row DailyMetric
	if err := db.Where(
		"metric_date = ? AND variant_id = ? AND source = ?",
		"2026-06-15",
		"b",
		"recommend",
	).First(&row).Error; err != nil {
		t.Fatalf("query daily metric: %v", err)
	}

	if row.ExposureCount != 1 ||
		row.ClickCount != 1 ||
		row.DwellCount != 1 ||
		row.DwellMsSum != 1500 ||
		row.LikeCount != 1 ||
		row.FavoriteCount != 1 ||
		row.CommentCount != 1 {
		t.Fatalf("daily metric row = %+v, want one count for each event", row)
	}
}

func TestDailyAggregatorUsesDefaultDimensions(t *testing.T) {
	db := newDailyMetricTestDB(t)
	aggregator := NewDailyAggregator(db)

	if err := aggregator.Aggregate(context.Background(), Event{
		EventType:  EventTypeClick,
		OccurredAt: time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC).Unix(),
	}); err != nil {
		t.Fatalf("Aggregate returned error: %v", err)
	}

	var row DailyMetric
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("query daily metric: %v", err)
	}
	if row.VariantID != "control" || row.Source != "unknown" || row.ClickCount != 1 {
		t.Fatalf("daily metric row = %+v, want default dimensions and click count", row)
	}
}

func newDailyMetricTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&DailyMetric{}); err != nil {
		t.Fatalf("migrate daily metric: %v", err)
	}
	return db
}
