package track

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	EventTypeExposure = "exposure"
	EventTypeClick    = "click"
	EventTypeDwell    = "dwell"
	EventTypeLike     = "like"
	EventTypeFavorite = "favorite"
	EventTypeComment  = "comment"
	EventTypeUnlike   = "unlike"
)

type Event struct {
	EventID    string  `json:"event_id"`
	EventType  string  `json:"event_type"`
	UserID     int64   `json:"user_id,omitempty"`
	ContentID  int64   `json:"content_id"`
	RequestID  string  `json:"request_id,omitempty"`
	SnapshotID string  `json:"snapshot_id"`
	VariantID  string  `json:"variant_id"`
	Source     string  `json:"source"`
	Position   int     `json:"position"`
	FinalScore float64 `json:"final_score,omitempty"`
	DwellMs    int64   `json:"dwell_ms,omitempty"`
	OccurredAt int64   `json:"occurred_at"`
}

func IsClientEventType(eventType string) bool {
	switch eventType {
	case EventTypeClick,
		EventTypeDwell,
		EventTypeLike,
		EventTypeFavorite,
		EventTypeComment,
		EventTypeUnlike:
		return true
	default:
		return false
	}
}

type Producer interface {
	Emit(ctx context.Context, event Event) error
}

type messagePusher interface {
	Push(ctx context.Context, value string) error
}

type KafkaProducer struct {
	pusher     messagePusher
	maxRetries int
}

type NoopProducer struct{}

func NewKafkaProducer(pusher messagePusher, maxRetries int) *KafkaProducer {
	if maxRetries <= 0 {
		maxRetries = 1
	}

	return &KafkaProducer{
		pusher:     pusher,
		maxRetries: maxRetries,
	}
}

func (p *KafkaProducer) Emit(ctx context.Context, event Event) error {
	if p == nil || p.pusher == nil {
		return errors.New("recommend track producer dependencies are not ready")
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal recommend track event: %w", err)
	}

	var lastErr error
	for i := 0; i < p.maxRetries; i++ {
		if err := p.pusher.Push(ctx, string(body)); err == nil {
			return nil
		} else {
			lastErr = err
			time.Sleep(time.Millisecond * 50 * time.Duration(i+1))
		}
	}
	return fmt.Errorf("push recommend track event: %w", lastErr)
}

func (NoopProducer) Emit(context.Context, Event) error {
	return nil
}
