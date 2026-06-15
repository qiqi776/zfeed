package track

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultDailyMetricVariant = "control"
	defaultDailyMetricSource  = "unknown"
)

type DailyMetric struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	MetricDate    string    `gorm:"column:metric_date;size:10;uniqueIndex:uk_rec_metric_day_variant_source,priority:1"`
	VariantID     string    `gorm:"column:variant_id;size:64;uniqueIndex:uk_rec_metric_day_variant_source,priority:2"`
	Source        string    `gorm:"column:source;size:32;uniqueIndex:uk_rec_metric_day_variant_source,priority:3"`
	ExposureCount int64     `gorm:"column:exposure_count"`
	ClickCount    int64     `gorm:"column:click_count"`
	DwellCount    int64     `gorm:"column:dwell_count"`
	DwellMsSum    int64     `gorm:"column:dwell_ms_sum"`
	LikeCount     int64     `gorm:"column:like_count"`
	FavoriteCount int64     `gorm:"column:favorite_count"`
	CommentCount  int64     `gorm:"column:comment_count"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (DailyMetric) TableName() string {
	return "zfeed_rec_metric_daily"
}

type DailyAggregator struct {
	db *gorm.DB
}

func NewDailyAggregator(db *gorm.DB) *DailyAggregator {
	return &DailyAggregator{db: db}
}

func (a *DailyAggregator) Aggregate(ctx context.Context, event Event) error {
	if a == nil || a.db == nil {
		return nil
	}

	now := time.Now().UTC()
	row, ok := buildDailyMetricRow(event, now)
	if !ok {
		return nil
	}

	return a.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "metric_date"},
				{Name: "variant_id"},
				{Name: "source"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"exposure_count": gorm.Expr("exposure_count + ?", row.ExposureCount),
				"click_count":    gorm.Expr("click_count + ?", row.ClickCount),
				"dwell_count":    gorm.Expr("dwell_count + ?", row.DwellCount),
				"dwell_ms_sum":   gorm.Expr("dwell_ms_sum + ?", row.DwellMsSum),
				"like_count":     gorm.Expr("like_count + ?", row.LikeCount),
				"favorite_count": gorm.Expr("favorite_count + ?", row.FavoriteCount),
				"comment_count":  gorm.Expr("comment_count + ?", row.CommentCount),
				"updated_at":     now,
			}),
		}).
		Create(&row).Error
}

func buildDailyMetricRow(event Event, now time.Time) (DailyMetric, bool) {
	occurredAt := now
	if event.OccurredAt > 0 {
		occurredAt = time.Unix(event.OccurredAt, 0).UTC()
	}

	row := DailyMetric{
		MetricDate: occurredAt.Format(time.DateOnly),
		VariantID:  normalizeDailyMetricDimension(event.VariantID, defaultDailyMetricVariant, 64),
		Source:     normalizeDailyMetricDimension(event.Source, defaultDailyMetricSource, 32),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	switch event.EventType {
	case EventTypeExposure:
		row.ExposureCount = 1
	case EventTypeClick:
		row.ClickCount = 1
	case EventTypeDwell:
		row.DwellCount = 1
		if event.DwellMs > 0 {
			row.DwellMsSum = event.DwellMs
		}
	case EventTypeLike:
		row.LikeCount = 1
	case EventTypeFavorite:
		row.FavoriteCount = 1
	case EventTypeComment:
		row.CommentCount = 1
	default:
		return DailyMetric{}, false
	}

	return row, true
}

func normalizeDailyMetricDimension(value, fallback string, maxLen int) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}
	if maxLen > 0 && len(value) > maxLen {
		return value[:maxLen]
	}
	return value
}
